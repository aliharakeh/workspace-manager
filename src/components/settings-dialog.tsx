import { useState } from "react"
import { Link2Icon, NetworkIcon } from "lucide-react"
import { PortsPanel } from "@/components/ports-panel"
import { ReadyUrlPatternsPanel } from "@/components/ready-url-patterns-panel"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

type SettingsDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

type SettingsTab = "ports" | "ready-urls"

export function SettingsDialog({ open, onOpenChange }: SettingsDialogProps) {
  const [tab, setTab] = useState<SettingsTab>("ports")

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex h-[min(42rem,90vh)] max-h-[min(42rem,90vh)] flex-col gap-0 overflow-hidden p-0 sm:max-w-4xl">
        <DialogHeader className="shrink-0 border-b p-4 pr-12">
          <DialogTitle>Settings</DialogTitle>
          <DialogDescription>
            Manage local tools and how App Runner detects ready URLs from logs.
          </DialogDescription>
        </DialogHeader>

        <Tabs
          value={tab}
          onValueChange={(value) => {
            if (value === "ports" || value === "ready-urls") setTab(value)
          }}
          orientation="vertical"
          className="min-h-0 flex-1 gap-0"
        >
          <TabsList
            variant="line"
            className="h-auto w-48 shrink-0 flex-col items-stretch justify-start rounded-none border-r p-2"
          >
            <TabsTrigger value="ports" className="justify-start">
              <NetworkIcon data-icon="inline-start" />
              Listening ports
            </TabsTrigger>
            <TabsTrigger value="ready-urls" className="justify-start">
              <Link2Icon data-icon="inline-start" />
              Log URL patterns
            </TabsTrigger>
          </TabsList>

          <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden p-4">
            <TabsContent
              value="ports"
              className="mt-0 flex min-h-0 flex-1 flex-col data-hidden:hidden"
            >
              <PortsPanel active={open && tab === "ports"} />
            </TabsContent>
            <TabsContent
              value="ready-urls"
              className="mt-0 flex min-h-0 flex-1 flex-col data-hidden:hidden"
            >
              <ReadyUrlPatternsPanel active={open && tab === "ready-urls"} />
            </TabsContent>
          </div>
        </Tabs>
      </DialogContent>
    </Dialog>
  )
}
