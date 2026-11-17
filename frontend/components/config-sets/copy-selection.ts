import type { CopyParts } from "@/lib/types"

export type CategorySel = { all: boolean; items: string[] }

export type CopySel = {
  env: CategorySel
  templates: CategorySel
  run_configs: CategorySel
}

function blankCategory(): CategorySel {
  return { all: true, items: [] }
}

export function blankSel(): CopySel {
  return {
    env: blankCategory(),
    templates: blankCategory(),
    run_configs: blankCategory(),
  }
}

function categoryToPart(sel: CategorySel): boolean | string[] {
  if (sel.all) return true
  return sel.items.length > 0 ? sel.items : false
}

export function selToParts(sel: CopySel): CopyParts {
  return {
    env: categoryToPart(sel.env),
    templates: categoryToPart(sel.templates),
    dev: false,
    prod: false,
    build: false,
    setup: false,
    run_configs: categoryToPart(sel.run_configs),
  }
}

function isPartEnabled(part: boolean | string[] | number[] | undefined) {
  if (part === undefined || part === true) return true
  return Array.isArray(part) && part.length > 0
}

export function partsEnabled(parts: CopyParts): boolean {
  return (
    isPartEnabled(parts.env) ||
    isPartEnabled(parts.templates) ||
    isPartEnabled(parts.run_configs)
  )
}
