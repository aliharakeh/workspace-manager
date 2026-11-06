import { Checkbox } from "@/components/ui/checkbox"
import { Field, FieldLabel } from "@/components/ui/field"
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@/components/ui/input-group"
import { SearchIcon, XIcon } from "lucide-react"
import type { CategorySel } from "./copy-selection"
type CategoryItem = { id: string; title: string; sub?: string }
export function CategoryPicker({
  items,
  sel,
  onChange,
  query,
  onQueryChange,
}: {
  items: CategoryItem[]
  sel: CategorySel
  onChange: (sel: CategorySel) => void
  query: string
  onQueryChange: (query: string) => void
}) {
  const count = items.length
  const allChecked = count > 0 && (sel.all || sel.items.length === count)
  const q = query.trim().toLowerCase()
  const visible = q
    ? items.filter(
        (item) =>
          item.title.toLowerCase().includes(q) ||
          (item.sub ?? "").toLowerCase().includes(q)
      )
    : items
  return (
    <div className="flex min-w-0 flex-col gap-2">
      {count > 0 ? (
        <InputGroup className="w-full min-w-0">
          <InputGroupAddon align="inline-start">
            <SearchIcon />
          </InputGroupAddon>
          <InputGroupInput
            value={query}
            onChange={(e) => onQueryChange(e.target.value)}
            placeholder="Search…"
            aria-label="Search items to copy"
          />
          {query ? (
            <InputGroupAddon align="inline-end">
              <InputGroupButton
                size="icon-xs"
                aria-label="Clear search"
                onClick={() => onQueryChange("")}
              >
                <XIcon />
              </InputGroupButton>
            </InputGroupAddon>
          ) : null}
        </InputGroup>
      ) : null}
      <Field orientation="horizontal">
        <Checkbox
          id="category-select-all"
          checked={allChecked}
          disabled={count === 0}
          onCheckedChange={() =>
            onChange(
              allChecked ? { all: false, items: [] } : { all: true, items: [] }
            )
          }
        />
        <FieldLabel htmlFor="category-select-all">Select all</FieldLabel>
        <span className="text-xs text-muted-foreground">
          {count === 0
            ? "(none)"
            : sel.all
              ? `All ${count}`
              : `${sel.items.length} of ${count}`}
        </span>
      </Field>
      {count > 0 && !sel.all ? (
        <div className="flex max-h-52 min-w-0 flex-col gap-1 overflow-y-auto rounded-lg border p-2 text-sm">
          {visible.length === 0 ? (
            <p className="px-1 py-2 text-xs text-muted-foreground">
              No items match “{query.trim()}”.
            </p>
          ) : (
            visible.map((item, index) => {
              const checkboxId = `category-item-${index}`
              return (
                <Field
                  key={item.id}
                  orientation="horizontal"
                  className="min-w-0"
                >
                  <Checkbox
                    id={checkboxId}
                    checked={sel.items.includes(item.id)}
                    onCheckedChange={(checked) => {
                      const next = new Set(sel.items)
                      if (checked) next.add(item.id)
                      else next.delete(item.id)
                      onChange(
                        next.size === count
                          ? { all: true, items: [] }
                          : { all: false, items: [...next] }
                      )
                    }}
                  />
                  <FieldLabel
                    htmlFor={checkboxId}
                    className="min-w-0 truncate font-normal"
                    title={item.sub ?? item.title}
                  >
                    {item.title}
                  </FieldLabel>
                </Field>
              )
            })
          )}
        </div>
      ) : null}
    </div>
  )
}
