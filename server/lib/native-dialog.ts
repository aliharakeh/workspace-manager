import { isAbsolute, normalize, relative, resolve, sep } from "node:path"

export type NativePickResult =
  | { ok: true; path: string }
  | { ok: false; cancelled: true }
  | { ok: false; error: string }

function quotePwsh(value: string) {
  return `'${value.replace(/'/g, "''")}'`
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

  const proc = Bun.spawn(
    [
      "powershell",
      "-NoProfile",
      "-STA",
      "-ExecutionPolicy",
      "Bypass",
      "-Command",
      script,
    ],
    { stdout: "pipe", stderr: "pipe" }
  )
  const [stdout, stderr, code] = await Promise.all([
    new Response(proc.stdout).text(),
    new Response(proc.stderr).text(),
    proc.exited,
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
  const proc = Bun.spawn(["osascript", "-e", script], {
    stdout: "pipe",
    stderr: "pipe",
  })
  const [stdout, stderr, code] = await Promise.all([
    new Response(proc.stdout).text(),
    new Response(proc.stderr).text(),
    proc.exited,
  ])
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

  const zenity = Bun.spawn(["zenity", ...args], {
    stdout: "pipe",
    stderr: "pipe",
  })
  const [stdout, stderr, code] = await Promise.all([
    new Response(zenity.stdout).text(),
    new Response(zenity.stderr).text(),
    zenity.exited,
  ])

  // zenity: 0 = ok, 1 = cancel
  if (code === 1) return { ok: false, cancelled: true }
  if (code === 0) {
    const selected = stdout.trim()
    if (!selected) return { ok: false, cancelled: true }
    return { ok: true, path: selected }
  }

  const kArgs = ["--getopenfilename"]
  if (startDir) kArgs.push(startDir)
  else kArgs.push(".")
  const kdialog = Bun.spawn(["kdialog", ...kArgs], {
    stdout: "pipe",
    stderr: "pipe",
  })
  const [kOut, kErr, kCode] = await Promise.all([
    new Response(kdialog.stdout).text(),
    new Response(kdialog.stderr).text(),
    kdialog.exited,
  ])
  if (kCode === 0) {
    const selected = kOut.trim()
    if (!selected) return { ok: false, cancelled: true }
    return { ok: true, path: selected }
  }
  if (kCode === 1) return { ok: false, cancelled: true }

  return {
    ok: false,
    error:
      stderr.trim() ||
      kErr.trim() ||
      "No native file dialog available (install zenity or kdialog)",
  }
}

/** Open the OS native open-file dialog. Returns absolute path or cancel/error. */
export async function pickNativeFile(
  startDir?: string
): Promise<NativePickResult> {
  const dir =
    startDir && startDir.trim() ? resolve(startDir.trim()) : undefined

  if (process.platform === "win32") return pickFileWindows(dir)
  if (process.platform === "darwin") return pickFileMac(dir)
  return pickFileLinux(dir)
}

/** Make `absolutePath` relative to `rootDir`, or null if outside root. */
export function toProjectRelative(
  rootDir: string,
  absolutePath: string
): string | null {
  const root = normalize(resolve(rootDir))
  const full = normalize(resolve(absolutePath))
  const rel = relative(root, full)
  if (!rel || isAbsolute(rel) || rel === ".." || rel.startsWith(`..${sep}`)) {
    return null
  }
  return rel.replace(/\\/g, "/")
}
