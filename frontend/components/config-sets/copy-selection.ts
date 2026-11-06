import type { CopyParts } from "@/lib/types"

export type CategorySel = { all: boolean; items: string[] }

export type CopySel = {
  env: CategorySel
  templates: CategorySel
  dev: CategorySel
  prod: CategorySel
  build: CategorySel
  setup: CategorySel
}

function blankCategory(): CategorySel {
  return { all: true, items: [] }
}

export function blankSel(): CopySel {
  return {
    env: blankCategory(),
    templates: blankCategory(),
    dev: blankCategory(),
    prod: blankCategory(),
    build: blankCategory(),
    setup: blankCategory(),
  }
}

function categoryToPart(sel: CategorySel): boolean | string[] {
  if (sel.all) return true
  return sel.items.length > 0 ? sel.items : false
}

function runToPart(sel: CategorySel): boolean | number[] {
  if (sel.all) return true
  return sel.items.length > 0 ? sel.items.map(Number) : false
}

export function selToParts(sel: CopySel): CopyParts {
  return {
    env: categoryToPart(sel.env),
    templates: categoryToPart(sel.templates),
    dev: runToPart(sel.dev),
    prod: runToPart(sel.prod),
    build: runToPart(sel.build),
    setup: runToPart(sel.setup),
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
    isPartEnabled(parts.dev) ||
    isPartEnabled(parts.prod) ||
    isPartEnabled(parts.build) ||
    isPartEnabled(parts.setup)
  )
}
