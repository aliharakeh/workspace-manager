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
import { Checkbox } from '@/components/ui/checkbox'
import {
    InputGroup,
    InputGroupAddon,
    InputGroupButton,
    InputGroupInput,
} from '@/components/ui/input-group'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { api, onDataChanged } from '@/lib/api'
import type { CommandKind, PackageScript, RunMode, StatusEvent, RunConfig } from '@/lib/types'
import {
    ChevronDownIcon,
    PlayIcon,
    SquareIcon,
    PlusIcon,
    SearchIcon,
    TerminalIcon,
    Trash2Icon,
    XIcon,
} from 'lucide-react'
import { useCallback, useDeferredValue, useEffect, useMemo, useState } from 'react'
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

type CommandConfigPanelProps = {
    appId: number
    /** Which command list to edit. */
    kind: CommandKind
    running: boolean
    anyRunning: boolean
    onDirtyChange: (dirty: boolean) => void
    onStatus: (status: StatusEvent) => void
}

/** Editor for a command list: an execution mode and its commands. */
export function CommandConfigPanel({
    appId,
    kind,
    running,
    anyRunning,
    onDirtyChange,
    onStatus,
}: CommandConfigPanelProps) {
    const noun = 'run'
    const placeholder = 'e.g. npm run dev'
    const [name, setName] = useState('')
    const [detectReady, setDetectReady] = useState(false)
    const [original, setOriginal] = useState('')
    const [mode, setMode] = useState<RunMode>('parallel')
    const [commands, setCommands] = useState<DraftCommand[]>([])
    const [packageScripts, setPackageScripts] = useState<PackageScript[]>([])
    const [query, setQuery] = useState('')
    const deferredQuery = useDeferredValue(query)
    const [loading, setLoading] = useState(true)
    const [saving, setSaving] = useState(false)
    const [loadError, setLoadError] = useState<string | null>(null)
    const [pendingRemove, setPendingRemove] = useState<DraftCommand | null>(null)

    const filtered = useMemo(() => {
        const q = deferredQuery.trim().toLowerCase()
        if (!q) return commands
        return commands.filter(
            c => c.label.toLowerCase().includes(q) || c.command.toLowerCase().includes(q),
        )
    }, [commands, deferredQuery])

    const acceptConfig = useCallback((config: RunConfig) => {
        setName(config.name || config.kind)
        setDetectReady(config.detect_ready)
        setMode(config.mode)
        setCommands(
            config.commands.map(c => ({
                key: String(c.id),
                label: c.label ?? '',
                command: c.command,
            })),
        )
        setOriginal(
            JSON.stringify({
                name: config.name || config.kind,
                detectReady: config.detect_ready,
                mode: config.mode,
                commands: config.commands.map(c => ({ label: c.label ?? '', command: c.command })),
            }),
        )
    }, [])

    useEffect(() => {
        let cancelled = false
        void api.runConfig
            .get(appId, kind)
            .then(config => {
                if (!cancelled) acceptConfig(config)
            })
            .catch(err => {
                if (!cancelled)
                    setLoadError(
                        err instanceof Error ? err.message : 'Failed to load configuration',
                    )
            })
            .finally(() => {
                if (!cancelled) setLoading(false)
            })
        return () => {
            cancelled = true
        }
    }, [appId, kind, acceptConfig])

    useEffect(() => {
        let cancelled = false
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

    const draft = JSON.stringify({
        name,
        detectReady,
        mode,
        commands: commands.map(c => ({ label: c.label, command: c.command })),
    })
    useEffect(() => {
        onDirtyChange(!loading && draft !== original)
    }, [draft, original, loading, onDirtyChange])

    useEffect(() => {
        if (loading || draft !== original) return
        let cancelled = false
        const off = onDataChanged(event => {
            if (event.entity !== 'runConfig' || !(event.ids ?? []).includes(appId)) return
            void api.runConfig
                .get(appId, kind)
                .then(config => {
                    if (!cancelled) acceptConfig(config)
                })
                .catch(err => {
                    if (!cancelled)
                        toast.error(
                            err instanceof Error ? err.message : 'Failed to refresh configuration',
                        )
                })
        })
        return () => {
            cancelled = true
            off()
        }
    }, [appId, kind, loading, draft, original, acceptConfig])

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

    async function handleSave(run = false) {
        if (!name.trim()) {
            toast.error('Give the configuration a name')
            return
        }
        if (commands.some(c => !c.command.trim())) {
            toast.error('Each process needs a command')
            return
        }
        setSaving(true)
        try {
            const saved = await api.runConfig.save(appId, kind, {
                name: name.trim(),
                detect_ready: detectReady,
                mode,
                commands: commands.map(c => ({
                    label: c.label.trim() || null,
                    command: c.command.trim(),
                })),
            })
            acceptConfig(saved)
            if (run) onStatus(await api.runner.start(appId, kind))
            else toast.success('Run configuration saved')
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to save')
        } finally {
            setSaving(false)
        }
    }

    if (loading) {
        return <p className="text-sm text-muted-foreground">Loading…</p>
    }

    if (loadError)
        return (
            <p role="alert" className="text-sm text-destructive">
                {loadError}
            </p>
        )

    const pendingName = pendingRemove?.label.trim() || pendingRemove?.command.trim()

    return (
        <div className="flex flex-col gap-4">
            {running ? (
                <div className="flex items-center gap-3">
                    <p role="status" className="text-sm">
                        Running {name}
                    </p>
                    <Button
                        variant="stop"
                        disabled={saving}
                        onClick={async () => {
                            setSaving(true)
                            try {
                                onStatus(await api.runner.stop(appId))
                            } catch (e) {
                                toast.error(e instanceof Error ? e.message : 'Failed to stop')
                            } finally {
                                setSaving(false)
                            }
                        }}
                    >
                        <SquareIcon data-icon="inline-start" className="fill-current" />
                        Stop
                    </Button>
                </div>
            ) : null}
            <fieldset disabled={running || saving} className="flex min-w-0 flex-col gap-4">
                <legend className="sr-only">Configuration settings</legend>
                <Field>
                    <FieldLabel htmlFor="run-config-name">Name</FieldLabel>
                    <Input
                        id="run-config-name"
                        name="name"
                        required
                        maxLength={128}
                        value={name}
                        onChange={e => setName(e.target.value)}
                    />
                </Field>
                <Field orientation="horizontal">
                    <Checkbox
                        id="run-detect-ready"
                        checked={detectReady}
                        onCheckedChange={checked => setDetectReady(checked === true)}
                    />
                    <FieldLabel htmlFor="run-detect-ready">
                        Detect ready URLs and idle activity
                    </FieldLabel>
                </Field>
                <Field>
                    <FieldLabel>Execution mode</FieldLabel>
                    <ToggleGroup
                        value={[mode]}
                        onValueChange={v => {
                            if (v[0] === 'sequential' || v[0] === 'parallel') setMode(v[0])
                        }}
                        aria-label="Execution mode"
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
                            {commands.length === 0
                                ? 'Add a command below to get started.'
                                : `No commands match “${query.trim()}”.`}
                        </p>
                    ) : (
                        filtered.map((cmd, index) => (
                            <div
                                key={cmd.key}
                                className="flex flex-col gap-2 rounded-lg border p-3 sm:flex-row sm:items-center"
                            >
                                <span className="w-6 text-xs text-muted-foreground">
                                    {index + 1}
                                </span>
                                <Input
                                    className="sm:max-w-40"
                                    aria-label={`Command ${index + 1} label`}
                                    placeholder="Label"
                                    value={cmd.label}
                                    onChange={e =>
                                        setCommands(prev =>
                                            prev.map(c =>
                                                c.key === cmd.key
                                                    ? { ...c, label: e.target.value }
                                                    : c,
                                            ),
                                        )
                                    }
                                />
                                <Input
                                    className="flex-1 font-mono"
                                    aria-label={`Command ${index + 1}`}
                                    spellCheck={false}
                                    autoCapitalize="none"
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
                        {saving ? 'Saving…' : 'Save configuration'}
                    </Button>
                </div>
            </fieldset>
            {!running ? (
                <Button
                    variant="run"
                    className="self-start"
                    disabled={saving || anyRunning || commands.length === 0}
                    onClick={() => void handleSave(true)}
                >
                    <PlayIcon data-icon="inline-start" className="fill-current" />
                    Save & run
                </Button>
            ) : null}
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
