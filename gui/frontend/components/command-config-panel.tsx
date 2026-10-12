import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuGroup,
    DropdownMenuItem,
    DropdownMenuLabel,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Field, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
    InputGroup,
    InputGroupAddon,
    InputGroupButton,
    InputGroupInput,
} from '@/components/ui/input-group'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { api } from '@/lib/api'
import type { CommandKind, PackageScript, RunMode, StatusEvent } from '@/lib/types'
import {
    ChevronDownIcon,
    PlayIcon,
    PlusIcon,
    SearchIcon,
    SquareIcon,
    TerminalIcon,
    Trash2Icon,
    XIcon,
} from 'lucide-react'
import { useDeferredValue, useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'

type DraftCommand = {
    key: string
    label: string
    command: string
}

function newKey() {
    return `new-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Menu items listing the project's package.json scripts. */
function ScriptMenuItems({
    scripts,
    onSelect,
}: {
    scripts: PackageScript[]
    onSelect: (script: PackageScript) => void
}) {
    return (
        <DropdownMenuGroup>
            <DropdownMenuLabel>package.json scripts</DropdownMenuLabel>
            {scripts.map(script => (
                <DropdownMenuItem key={script.name} onClick={() => onSelect(script)}>
                    <span className="flex min-w-0 flex-col">
                        <span className="truncate font-mono text-xs">{script.name}</span>
                        <span className="truncate text-xs text-muted-foreground">
                            {script.script}
                        </span>
                    </span>
                </DropdownMenuItem>
            ))}
        </DropdownMenuGroup>
    )
}

const KIND_COPY: Record<CommandKind, { noun: string; placeholder: string }> = {
    run: { noun: 'run', placeholder: 'npm run dev' },
    build: { noun: 'build', placeholder: 'npm run build' },
    setup: { noun: 'setup', placeholder: 'npm install' },
}

type CommandConfigPanelProps = {
    appId: number
    /** Which command list to edit: the run, build or setup config. */
    kind: CommandKind
    /**
     * The app's runner status and its handler. With both, a "setup" panel gets
     * its own Run/Stop button, so setup is run in place and needs no button
     * elsewhere.
     */
    status?: StatusEvent | null
    onStatus?: (status: StatusEvent) => void
}

/** Editor for an app's run, build or setup config: an execution mode and its commands. */
export function CommandConfigPanel({ appId, kind, status, onStatus }: CommandConfigPanelProps) {
    const { noun, placeholder } = KIND_COPY[kind]
    const [mode, setMode] = useState<RunMode>('parallel')
    const [commands, setCommands] = useState<DraftCommand[]>([])
    const [packageScripts, setPackageScripts] = useState<PackageScript[]>([])
    const [query, setQuery] = useState('')
    const deferredQuery = useDeferredValue(query)
    const [loading, setLoading] = useState(true)
    const [saving, setSaving] = useState(false)
    const [starting, setStarting] = useState(false)
    const [pendingRemove, setPendingRemove] = useState<DraftCommand | null>(null)

    const filtered = useMemo(() => {
        const q = deferredQuery.trim().toLowerCase()
        if (!q) return commands
        return commands.filter(
            c => c.label.toLowerCase().includes(q) || c.command.toLowerCase().includes(q),
        )
    }, [commands, deferredQuery])

    useEffect(() => {
        let cancelled = false
        setPendingRemove(null)
        ;(async () => {
            setLoading(true)
            try {
                const config = await api.runConfig.get(appId, kind)
                if (cancelled) return
                setMode(config.mode)
                setCommands(
                    config.commands.map(c => ({
                        key: String(c.id),
                        label: c.label ?? '',
                        command: c.command,
                    })),
                )
            } catch (err) {
                toast.error(err instanceof Error ? err.message : `Failed to load ${noun} config`)
            } finally {
                if (!cancelled) setLoading(false)
            }
        })()
        return () => {
            cancelled = true
        }
    }, [appId, kind])

    useEffect(() => {
        let cancelled = false
        setPackageScripts([])
        void (async () => {
            try {
                const result = await api.packageScripts.list(appId)
                if (!cancelled) setPackageScripts(result.scripts ?? [])
            } catch {
                // The picker is a shortcut only — manual entry always works.
            }
        })()
        return () => {
            cancelled = true
        }
    }, [appId])

    function addCommand(command = '', label = '') {
        setCommands(prev => [...prev, { key: newKey(), label, command }])
    }

    function applyScript(key: string, script: PackageScript) {
        setCommands(prev =>
            prev.map(c =>
                c.key === key
                    ? {
                          ...c,
                          command: script.command,
                          label: c.label.trim() ? c.label : script.name,
                      }
                    : c,
            ),
        )
    }

    /** Saves the draft; resolves to whether it was saved. */
    async function handleSave(quiet = false): Promise<boolean> {
        if (commands.some(c => !c.command.trim())) {
            toast.error('Each process needs a command')
            return false
        }
        setSaving(true)
        try {
            const saved = await api.runConfig.save(appId, kind, {
                mode,
                commands: commands.map(c => ({
                    label: c.label.trim() || null,
                    command: c.command.trim(),
                })),
            })
            setMode(saved.mode)
            setCommands(
                saved.commands.map(c => ({
                    key: String(c.id),
                    label: c.label ?? '',
                    command: c.command,
                })),
            )
            if (!quiet) toast.success(`${noun[0]!.toUpperCase()}${noun.slice(1)} config saved`)
            return true
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to save')
            return false
        } finally {
            setSaving(false)
        }
    }

    // The runner reads the saved config, so the draft is saved before it starts.
    async function handleRunSetup() {
        if (!onStatus || commands.length === 0) return
        setStarting(true)
        try {
            if (!(await handleSave(true))) return
            onStatus(await api.runner.setup(appId))
            toast.success('Setup started')
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to run setup')
        } finally {
            setStarting(false)
        }
    }

    async function handleStopSetup() {
        if (!onStatus) return
        setStarting(true)
        try {
            onStatus(await api.runner.stop(appId))
            toast.success('Stopped')
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to stop')
        } finally {
            setStarting(false)
        }
    }

    if (loading) {
        return <p className="text-sm text-muted-foreground">Loading…</p>
    }

    const pendingName = pendingRemove?.label.trim() || pendingRemove?.command.trim()
    const settingUp = !!status?.running && status.kind === 'setup'
    const otherSessionRunning = !!status?.running && status.kind !== 'setup'

    return (
        <div className="flex flex-col gap-4">
            <Field>
                <FieldLabel>Execution mode</FieldLabel>
                <ToggleGroup
                    value={[mode]}
                    onValueChange={v => {
                        if (v[0] === 'sequential' || v[0] === 'parallel') setMode(v[0])
                    }}
                    spacing={2}
                >
                    <ToggleGroupItem value="parallel">Parallel</ToggleGroupItem>
                    <ToggleGroupItem value="sequential">Sequential</ToggleGroupItem>
                </ToggleGroup>
                <p className="text-xs text-muted-foreground">
                    {mode === 'parallel'
                        ? 'All commands start together.'
                        : 'Commands run one after another; a non-zero exit stops the chain.'}
                </p>
            </Field>

            {commands.length > 0 ? (
                <div className="flex items-center gap-2">
                    <InputGroup className="max-w-sm flex-1">
                        <InputGroupAddon align="inline-start">
                            <SearchIcon />
                        </InputGroupAddon>
                        <InputGroupInput
                            value={query}
                            onChange={e => setQuery(e.target.value)}
                            placeholder="Search label or command…"
                            aria-label="Search commands"
                        />
                        {query ? (
                            <InputGroupAddon align="inline-end">
                                <InputGroupButton
                                    size="icon-xs"
                                    aria-label="Clear search"
                                    onClick={() => setQuery('')}
                                >
                                    <XIcon />
                                </InputGroupButton>
                            </InputGroupAddon>
                        ) : null}
                    </InputGroup>
                    <span className="text-xs text-muted-foreground">
                        {filtered.length} of {commands.length}
                    </span>
                </div>
            ) : null}

            <div className="flex flex-col gap-2">
                {filtered.length === 0 ? (
                    <p className="text-sm text-muted-foreground">
                        No commands match “{query.trim()}”.
                    </p>
                ) : (
                    filtered.map((cmd, index) => (
                        <div
                            key={cmd.key}
                            className="flex flex-col gap-2 rounded-lg border p-3 sm:flex-row sm:items-center"
                        >
                            <span className="w-6 text-xs text-muted-foreground">{index + 1}</span>
                            <Input
                                className="sm:max-w-40"
                                placeholder="Label"
                                value={cmd.label}
                                onChange={e =>
                                    setCommands(prev =>
                                        prev.map(c =>
                                            c.key === cmd.key ? { ...c, label: e.target.value } : c,
                                        ),
                                    )
                                }
                            />
                            <Input
                                className="flex-1 font-mono"
                                placeholder={placeholder}
                                value={cmd.command}
                                onChange={e =>
                                    setCommands(prev =>
                                        prev.map(c =>
                                            c.key === cmd.key
                                                ? { ...c, command: e.target.value }
                                                : c,
                                        ),
                                    )
                                }
                            />
                            {packageScripts.length > 0 ? (
                                <DropdownMenu>
                                    <DropdownMenuTrigger
                                        render={
                                            <Button
                                                variant="ghost"
                                                size="icon"
                                                aria-label="Use a package.json script"
                                            />
                                        }
                                    >
                                        <TerminalIcon />
                                    </DropdownMenuTrigger>
                                    <DropdownMenuContent align="end" className="min-w-72">
                                        <ScriptMenuItems
                                            scripts={packageScripts}
                                            onSelect={script => applyScript(cmd.key, script)}
                                        />
                                    </DropdownMenuContent>
                                </DropdownMenu>
                            ) : null}
                            <Button
                                variant="ghost"
                                size="icon"
                                onClick={() => setPendingRemove(cmd)}
                                aria-label="Remove command"
                            >
                                <Trash2Icon />
                            </Button>
                        </div>
                    ))
                )}
            </div>

            <div className="flex flex-wrap gap-2">
                <Button variant="outline" onClick={() => addCommand()}>
                    <PlusIcon data-icon="inline-start" />
                    Add command
                </Button>
                {packageScripts.length > 0 ? (
                    <DropdownMenu>
                        <DropdownMenuTrigger render={<Button variant="outline" />}>
                            <TerminalIcon data-icon="inline-start" />
                            Add from package.json
                            <ChevronDownIcon data-icon="inline-end" />
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="start" className="min-w-72">
                            <ScriptMenuItems
                                scripts={packageScripts}
                                onSelect={script => addCommand(script.command, script.name)}
                            />
                        </DropdownMenuContent>
                    </DropdownMenu>
                ) : null}
                <Button disabled={saving} onClick={() => void handleSave()}>
                    {saving ? 'Saving…' : `Save ${noun} config`}
                </Button>
                {kind === 'setup' && onStatus ? (
                    settingUp ? (
                        <Button
                            variant="destructive"
                            disabled={starting}
                            onClick={() => void handleStopSetup()}
                        >
                            <SquareIcon data-icon="inline-start" />
                            Stop setup
                        </Button>
                    ) : (
                        <Button
                            className="bg-emerald-600 text-white hover:bg-emerald-700"
                            disabled={starting || saving || otherSessionRunning || commands.length === 0}
                            title={
                                otherSessionRunning
                                    ? 'Stop the running app first'
                                    : commands.length === 0
                                      ? 'Add a setup command first'
                                      : 'Save and run the setup commands'
                            }
                            onClick={() => void handleRunSetup()}
                        >
                            <PlayIcon data-icon="inline-start" />
                            {starting ? 'Starting…' : 'Run setup'}
                        </Button>
                    )
                ) : null}
            </div>

            <AlertDialog
                open={!!pendingRemove}
                onOpenChange={next => {
                    if (!next) setPendingRemove(null)
                }}
            >
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Remove command?</AlertDialogTitle>
                        <AlertDialogDescription>
                            {pendingName
                                ? `This removes “${pendingName}” from the ${noun} config. Save to keep the change.`
                                : `This removes the command from the ${noun} config. Save to keep the change.`}
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction
                            variant="destructive"
                            onClick={() => {
                                if (!pendingRemove) return
                                setCommands(prev => prev.filter(c => c.key !== pendingRemove.key))
                                setPendingRemove(null)
                            }}
                        >
                            Remove
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
    )
}
