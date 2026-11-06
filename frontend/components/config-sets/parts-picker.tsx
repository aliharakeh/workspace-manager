import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import type { ConfigSetDetail } from "@/lib/types"
import { useState } from "react"
import { CategoryPicker } from "./category-picker"
import type { CopySel } from "./copy-selection"
export function PartsPicker({
  detail,
  loading,
  error,
  sel,
  onChange,
}: {
  detail: ConfigSetDetail | null
  loading: boolean
  error: string | null
  sel: CopySel
  onChange: (sel: CopySel) => void
}) {
  const [tab, setTab] = useState("env")
  const [query, setQuery] = useState("")

  if (loading) {
    return <p className="text-xs text-muted-foreground">Loading source set…</p>
  }
  if (error) {
    return <p className="text-xs text-destructive">{error}</p>
  }
  if (!detail) return null
  return (
    <Tabs
      value={tab}
      onValueChange={(value) => {
        if (value) {
          setTab(value)
          setQuery("")
        }
      }}
    >
      <TabsList className="w-full">
        <TabsTrigger value="env" className="flex-1">
          Env vars
        </TabsTrigger>
        <TabsTrigger value="templates" className="flex-1">
          Templates
        </TabsTrigger>
        <TabsTrigger value="setup" className="flex-1">
          Setup config
        </TabsTrigger>
        <TabsTrigger value="dev" className="flex-1">
          Dev config
        </TabsTrigger>
        <TabsTrigger value="build" className="flex-1">
          Build config
        </TabsTrigger>
        <TabsTrigger value="prod" className="flex-1">
          Prod config
        </TabsTrigger>
      </TabsList>
      <TabsContent value="env" className="mt-3">
        <CategoryPicker
          items={detail.env_vars.map((v) => ({
            id: v.key,
            title: v.key,
            sub: v.value,
          }))}
          sel={sel.env}
          onChange={(env) => onChange({ ...sel, env })}
          query={query}
          onQueryChange={setQuery}
        />
      </TabsContent>
      <TabsContent value="templates" className="mt-3">
        <CategoryPicker
          items={detail.templates.map((t) => ({
            id: t.file_path,
            title: t.file_path,
          }))}
          sel={sel.templates}
          onChange={(templates) => onChange({ ...sel, templates })}
          query={query}
          onQueryChange={setQuery}
        />
      </TabsContent>
      <TabsContent value="setup" className="mt-3">
        <CategoryPicker
          items={(detail.setup_config?.commands ?? []).map((c) => ({
            id: String(c.id),
            title: c.label ?? c.command,
            sub: c.label ? c.command : undefined,
          }))}
          sel={sel.setup}
          onChange={(setup) => onChange({ ...sel, setup })}
          query={query}
          onQueryChange={setQuery}
        />
      </TabsContent>
      <TabsContent value="dev" className="mt-3">
        <CategoryPicker
          items={(detail.dev_config?.commands ?? []).map((c) => ({
            id: String(c.id),
            title: c.label ?? c.command,
            sub: c.label ? c.command : undefined,
          }))}
          sel={sel.dev}
          onChange={(dev) => onChange({ ...sel, dev })}
          query={query}
          onQueryChange={setQuery}
        />
      </TabsContent>
      <TabsContent value="build" className="mt-3">
        <CategoryPicker
          items={(detail.build_config?.commands ?? []).map((c) => ({
            id: String(c.id),
            title: c.label ?? c.command,
            sub: c.label ? c.command : undefined,
          }))}
          sel={sel.build}
          onChange={(build) => onChange({ ...sel, build })}
          query={query}
          onQueryChange={setQuery}
        />
      </TabsContent>
      <TabsContent value="prod" className="mt-3">
        <CategoryPicker
          items={(detail.prod_config?.commands ?? []).map((c) => ({
            id: String(c.id),
            title: c.label ?? c.command,
            sub: c.label ? c.command : undefined,
          }))}
          sel={sel.prod}
          onChange={(prod) => onChange({ ...sel, prod })}
          query={query}
          onQueryChange={setQuery}
        />
      </TabsContent>
    </Tabs>
  )
}
