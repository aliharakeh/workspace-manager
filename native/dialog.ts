/**
 * Cross-platform native file/folder picker dialogs.
 *
 * Windows: PowerShell + WinForms
 * macOS: osascript
 * Linux: zenity, then kdialog
 */

import { statSync } from "node:fs"
import { resolve, sep } from "node:path"
import { run } from "./run"

export type NativePickResult =
  | { ok: true; path: string }
  | { ok: false; cancelled: true }
  | { ok: false; error: string }

function quotePwsh(value: string) {
  return `'${value.replace(/'/g, "''")}'`
}

/** Absolute existing directory, or undefined so the picker still opens. */
function existingDir(startDir?: string): string | undefined {
  if (!startDir?.trim()) return undefined
  try {
    const dir = resolve(startDir.trim())
    if (statSync(dir).isDirectory()) return dir
  } catch {
    // missing / inaccessible
  }
  return undefined
}

async function pickFileWindows(startDir?: string): Promise<NativePickResult> {
  const initial = startDir
    ? quotePwsh(startDir)
    : '([Environment]::GetFolderPath("MyDocuments"))'
  const script = `
Add-Type -AssemblyName System.Windows.Forms | Out-Null
$dialog = New-Object System.Windows.Forms.OpenFileDialog
$dialog.Title = "Select template file"
$dialog.Filter = "All files (*.*)|*.*"
$dialog.Multiselect = $false
$dialog.CheckFileExists = $true
$dialog.InitialDirectory = ${initial}
$dialog.RestoreDirectory = $true
$result = $dialog.ShowDialog()
if ($result -eq [System.Windows.Forms.DialogResult]::OK) {
  [Console]::Out.Write($dialog.FileName)
} else {
  [Console]::Out.Write("")
}
`.trim()

  const { stdout, stderr, code } = await run([
    "powershell",
    "-NoProfile",
    "-STA",
    "-ExecutionPolicy",
    "Bypass",
    "-Command",
    script,
  ])
  if (code !== 0) {
    return {
      ok: false,
      error: stderr.trim() || `File dialog failed (exit ${code})`,
    }
  }
  const selected = stdout.trim()
  if (!selected) return { ok: false, cancelled: true }
  return { ok: true, path: selected }
}

async function pickFileMac(startDir?: string): Promise<NativePickResult> {
  const location = startDir
    ? ` default location (POSIX file ${JSON.stringify(startDir)})`
    : ""
  const script = `try
  set f to choose file with prompt "Select template file"${location}
  return POSIX path of f
on error number -128
  return ""
end try`
  const { stdout, stderr, code } = await run(["osascript", "-e", script])
  const selected = stdout.trim()
  if (!selected) {
    if (code !== 0 && stderr.trim()) {
      return { ok: false, error: stderr.trim() }
    }
    return { ok: false, cancelled: true }
  }
  return { ok: true, path: selected }
}

async function pickFileLinux(startDir?: string): Promise<NativePickResult> {
  const args = ["--file-selection", "--title=Select template file"]
  if (startDir) args.push(`--filename=${startDir}${sep}`)

  const zenity = await run(["zenity", ...args])

  // zenity: 0 = ok, 1 = cancel
  if (zenity.code === 1) return { ok: false, cancelled: true }
  if (zenity.code === 0) {
    const selected = zenity.stdout.trim()
    if (!selected) return { ok: false, cancelled: true }
    return { ok: true, path: selected }
  }

  const kArgs = ["--getopenfilename"]
  if (startDir) kArgs.push(startDir)
  else kArgs.push(".")
  const kdialog = await run(["kdialog", ...kArgs])
  if (kdialog.code === 0) {
    const selected = kdialog.stdout.trim()
    if (!selected) return { ok: false, cancelled: true }
    return { ok: true, path: selected }
  }
  if (kdialog.code === 1) return { ok: false, cancelled: true }

  return {
    ok: false,
    error:
      zenity.stderr.trim() ||
      kdialog.stderr.trim() ||
      "No native file dialog available (install zenity or kdialog)",
  }
}

/** Open the OS native open-file dialog. Returns absolute path or cancel/error. */
export async function pickNativeFile(
  startDir?: string
): Promise<NativePickResult> {
  const dir = existingDir(startDir)

  if (process.platform === "win32") return pickFileWindows(dir)
  if (process.platform === "darwin") return pickFileMac(dir)
  return pickFileLinux(dir)
}

async function pickFolderWindows(startDir?: string): Promise<NativePickResult> {
  const initial = startDir
    ? quotePwsh(startDir)
    : '([Environment]::GetFolderPath("MyDocuments"))'
  const script = `
Add-Type -AssemblyName System.Windows.Forms | Out-Null
$dialog = New-Object System.Windows.Forms.FolderBrowserDialog
$dialog.Description = "Select project folder"
$dialog.ShowNewFolderButton = $true
$dialog.RootFolder = [Environment+SpecialFolder]::MyComputer
$dialog.SelectedPath = ${initial}
$owner = New-Object System.Windows.Forms.Form
$owner.TopMost = $true
$owner.ShowInTaskbar = $false
$owner.FormBorderStyle = 'None'
$owner.Size = New-Object System.Drawing.Size(1, 1)
$owner.StartPosition = 'Manual'
$owner.Location = New-Object System.Drawing.Point(-2000, -2000)
$null = $owner.Show()
try {
  $result = $dialog.ShowDialog($owner)
} finally {
  $owner.Dispose()
}
if ($result -eq [System.Windows.Forms.DialogResult]::OK) {
  [Console]::Out.Write($dialog.SelectedPath)
} else {
  [Console]::Out.Write("")
}
`.trim()

  const { stdout, stderr, code } = await run([
    "powershell",
    "-NoProfile",
    "-STA",
    "-ExecutionPolicy",
    "Bypass",
    "-Command",
    script,
  ])
  if (code !== 0) {
    return {
      ok: false,
      error: stderr.trim() || `Folder dialog failed (exit ${code})`,
    }
  }
  const selected = stdout.trim()
  if (!selected) return { ok: false, cancelled: true }
  return { ok: true, path: selected }
}

async function pickFolderMac(startDir?: string): Promise<NativePickResult> {
  const location = startDir
    ? ` default location (POSIX file ${JSON.stringify(startDir)})`
    : ""
  const script = `try
  set f to choose folder with prompt "Select project folder"${location}
  return POSIX path of f
on error number -128
  return ""
end try`
  const { stdout, stderr, code } = await run(["osascript", "-e", script])
  const selected = stdout.trim()
  if (!selected) {
    if (code !== 0 && stderr.trim()) {
      return { ok: false, error: stderr.trim() }
    }
    return { ok: false, cancelled: true }
  }
  return { ok: true, path: selected }
}

async function pickFolderLinux(startDir?: string): Promise<NativePickResult> {
  const args = [
    "--file-selection",
    "--directory",
    "--title=Select project folder",
  ]
  if (startDir) args.push(`--filename=${startDir}${sep}`)

  const zenity = await run(["zenity", ...args])

  if (zenity.code === 1) return { ok: false, cancelled: true }
  if (zenity.code === 0) {
    const selected = zenity.stdout.trim()
    if (!selected) return { ok: false, cancelled: true }
    return { ok: true, path: selected }
  }

  const kArgs = ["--getexistingdirectory"]
  if (startDir) kArgs.push(startDir)
  else kArgs.push(".")
  const kdialog = await run(["kdialog", ...kArgs])
  if (kdialog.code === 0) {
    const selected = kdialog.stdout.trim()
    if (!selected) return { ok: false, cancelled: true }
    return { ok: true, path: selected }
  }
  if (kdialog.code === 1) return { ok: false, cancelled: true }

  return {
    ok: false,
    error:
      zenity.stderr.trim() ||
      kdialog.stderr.trim() ||
      "No native folder dialog available (install zenity or kdialog)",
  }
}

/** Open the OS native folder dialog. Returns absolute path or cancel/error. */
export async function pickNativeFolder(
  startDir?: string
): Promise<NativePickResult> {
  const dir = existingDir(startDir)

  if (process.platform === "win32") return pickFolderWindows(dir)
  if (process.platform === "darwin") return pickFolderMac(dir)
  return pickFolderLinux(dir)
}
