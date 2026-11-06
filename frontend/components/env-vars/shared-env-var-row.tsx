import { SecretInput } from "@/components/secret-input"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import type { EnvVar } from "@/lib/types"
import { LockIcon } from "lucide-react"
/** A variable taken from the workspace's shared config: shown, never edited here. */
export function SharedEnvVarRow({ item }: { item: EnvVar }) {
  const isChoice = item.options.length > 0
  return (
    <div className="flex flex-col gap-2 rounded-lg border bg-muted/30 p-3 sm:flex-row sm:items-center">
      <div className="flex items-center gap-2 sm:w-48">
        <LockIcon
          className="size-3.5 shrink-0 text-muted-foreground"
          aria-label="Shared from the workspace"
        />
        <Input
          className="font-mono"
          value={item.key}
          readOnly
          aria-label="Variable key"
        />
      </div>
      <SecretInput
        secret={item.private}
        className="flex-1 font-mono"
        value={item.value}
        readOnly
        aria-label="Variable value"
        title={
          isChoice
            ? `Option “${item.selected}” of the shared variable`
            : undefined
        }
      />
      <div className="flex items-center gap-2">
        {isChoice ? <Badge variant="outline">{item.selected}</Badge> : null}
        {item.private ? (
          <Badge variant="outline" title="Never shown to the AI or to agents">
            private
          </Badge>
        ) : null}
      </div>
    </div>
  )
}
