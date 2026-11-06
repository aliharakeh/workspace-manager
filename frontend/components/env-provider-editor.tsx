import { useEffect, useId, useState } from "react"
import { api } from "@/lib/api"
import { Button } from "@/components/ui/button"
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogFooter,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { toast } from "sonner"

export function EnvProviderEditor({
  shared = false,
  setId,
}: {
  shared?: boolean
  setId: number
}) {
  const id = useId()
  const [open, setOpen] = useState(false)
  const [savedCommand, setSavedCommand] = useState("")
  const [command, setCommand] = useState("")
  const [directory, setDirectory] = useState("")
  const [timeout, setTimeoutValue] = useState(30)
  const [busy, setBusy] = useState(true)
  const [loaded, setLoaded] = useState(false)
  useEffect(() => {
    let active = true
    api.envProvider
      .get(shared, setId)
      .then((p) => {
        if (!active) return
        setCommand(p.command)
        setSavedCommand(p.command)
        setDirectory(p.directory)
        setTimeoutValue(p.timeout)
        setLoaded(true)
      })
      .catch((e) => {
        if (active) toast.error(String(e))
      })
      .finally(() => {
        if (active) setBusy(false)
      })
    return () => {
      active = false
    }
  }, [shared, setId])
  async function save() {
    setBusy(true)
    try {
      await api.envProvider.save(shared, setId, {
        command,
        directory,
        timeout,
      })
      setSavedCommand(command.trim())
      setOpen(false)
      toast.success("Dynamic environment saved")
    } catch (e) {
      toast.error(String(e))
    } finally {
      setBusy(false)
    }
  }
  async function runNow() {
    setBusy(true)
    try {
      await api.envProvider.run(shared, setId)
      toast.success(
        "Dynamic values refreshed and saved to the variables."
      )
    } catch (e) {
      toast.error(String(e))
    } finally {
      setBusy(false)
    }
  }
  function changeOpen(value: boolean) {
    if (busy) return
    setOpen(value)
    if (value) {
      setBusy(true)
      api.envProvider
        .get(shared, setId)
        .then((p) => {
          setCommand(p.command)
          setDirectory(p.directory)
          setTimeoutValue(p.timeout)
        })
        .catch((e) => toast.error(String(e)))
        .finally(() => setBusy(false))
    }
  }
  return (
    <div className="flex flex-wrap items-center gap-2">
      <Button
        variant="outline"
        disabled={busy || !loaded || !savedCommand}
        onClick={() => void runNow()}
      >
        {busy && loaded ? "Running…" : "Run script now"}
      </Button>
      <Dialog open={open} onOpenChange={changeOpen}>
        <DialogTrigger
          render={
            <Button variant="outline" disabled={busy || !loaded}>
              Configure script
            </Button>
          }
        />
        <DialogContent className="@container max-h-[85dvh] overflow-y-auto sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>Dynamic environment script</DialogTitle>
            <DialogDescription>
              Manually refresh {shared ? "workspace shared" : "app"} variables
              with a script. Save your settings, then choose Run script now.
            </DialogDescription>
          </DialogHeader>
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor={id + "command"}>Script command</FieldLabel>
              <Input
                id={id + "command"}
                value={command}
                disabled={busy || !loaded}
                onChange={(e) => setCommand(e.target.value)}
                placeholder="node scripts/runtime-env.mjs"
                aria-describedby={id + "command-help"}
              />
              <FieldDescription id={id + "command-help"}>
                Leave empty to disable the script. Use private variables for
                credentials.
              </FieldDescription>
            </Field>
            {shared && (
              <Field>
                <FieldLabel htmlFor={id + "directory"}>
                  Working directory
                </FieldLabel>
                <Input
                  id={id + "directory"}
                  value={directory}
                  disabled={busy || !loaded}
                  onChange={(e) => setDirectory(e.target.value)}
                  aria-describedby={id + "directory-help"}
                />
                <FieldDescription id={id + "directory-help"}>
                  Absolute path to the folder where the shared script runs.
                </FieldDescription>
              </Field>
            )}
            <Field>
              <FieldLabel htmlFor={id + "timeout"}>
                Timeout (seconds)
              </FieldLabel>
              <Input
                id={id + "timeout"}
                type="number"
                min={1}
                max={600}
                value={timeout}
                disabled={busy || !loaded}
                onChange={(e) => setTimeoutValue(Number(e.target.value))}
                aria-describedby={id + "timeout-help"}
              />
              <FieldDescription id={id + "timeout-help"}>
                1–600 seconds.
              </FieldDescription>
            </Field>

            <details className="rounded-lg border p-3">
              <summary className="cursor-pointer text-sm font-medium">
                Output format and usage
              </summary>
              <div className="mt-3 flex flex-col gap-3 text-sm text-muted-foreground">
                <p>
                  Print a JSON object of string values to stdout. All returned
                  keys are merged into the runtime environment; matching keys
                  are overwritten.
                </p>
                <pre className="overflow-x-auto rounded-md bg-muted p-3 text-xs text-foreground">
                  <code>
                    {
                      '{"API_TOKEN":"fresh-token","API_URL":"https://api.example.test"}'
                    }
                  </code>
                </pre>
                <p>
                  App starts use the latest successful result. Running apps keep
                  their current values. Run shared scripts before app scripts
                  that depend on them. App output overrides matching runtime
                  keys.
                </p>
                <p>
                  Results are cleared when the daemon restarts. Run the script
                  again afterward.
                </p>
              </div>
            </details>
          </FieldGroup>
          <DialogFooter>
            <Button
              variant="outline"
              disabled={busy}
              onClick={() => setOpen(false)}
            >
              Cancel
            </Button>
            <Button disabled={busy || !loaded} onClick={() => void save()}>
              {busy ? "Saving…" : "Save changes"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
