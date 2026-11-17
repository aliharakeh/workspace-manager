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
      className="@container w-full min-w-0"
      value={tab}
      onValueChange={(value) => {
        if (value) {
          setTab(value)
          setQuery("")
        }
      }}
    >
      <TabsList className="grid w-full grid-cols-2 group-data-horizontal/tabs:h-auto @sm:grid-cols-3 @2xl:grid-cols-3">
        <TabsTrigger value="env" className="h-8 min-w-0">
          Env vars
        </TabsTrigger>
        <TabsTrigger value="templates" className="h-8 min-w-0">
          Templates
        </TabsTrigger>
        <TabsTrigger value="run_configs" className="h-8 min-w-0">
          Run configurations
        </TabsTrigger>
      </TabsList>
      <TabsContent value="env" className="mt-3 min-w-0">
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
      <TabsContent value="templates" className="mt-3 min-w-0">
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
      <TabsContent value="run_configs" className="mt-3 min-w-0">
        <CategoryPicker
          items={detail.run_configs.map((c) => ({
            id: c.kind,
            title: c.name,
            sub: `${c.commands.length} commands · ${c.mode}`,
          }))}
          sel={sel.run_configs}
          onChange={(run_configs) => onChange({ ...sel, run_configs })}
          query={query}
          onQueryChange={setQuery}
        />
      </TabsContent>
    </Tabs>
  )
}
