import { asc, eq, sql } from "drizzle-orm"
import { db } from "./index"
import { readyUrlPatterns } from "./schema"
import type { ReadyUrlPatternRow } from "./types"
import { DEFAULT_READY_URL_PATTERNS } from "../services/ready-url-defaults"

export const readyUrlPatternsRepo = {
  list(): ReadyUrlPatternRow[] {
    return db
      .select()
      .from(readyUrlPatterns)
      .orderBy(asc(readyUrlPatterns.sort_order), asc(readyUrlPatterns.id))
      .all()
  },

  get(id: number): ReadyUrlPatternRow | null {
    return (
      db
        .select()
        .from(readyUrlPatterns)
        .where(eq(readyUrlPatterns.id, id))
        .get() ?? null
    )
  },

  /**
   * Persist built-in defaults into the DB.
   * - Empty table → insert all defaults
   * - Legacy rows (no keys) → attach keys / refresh labels by pattern match
   * - Missing default keys → insert them (app upgrades)
   */
  ensureSeeded(): void {
    const rows = this.list()

    if (rows.length === 0) {
      for (const [i, entry] of DEFAULT_READY_URL_PATTERNS.entries()) {
        db.insert(readyUrlPatterns)
          .values({
            key: entry.key,
            label: entry.label,
            pattern: entry.pattern,
            flags: entry.flags,
            sort_order: i,
          })
          .run()
      }
      return
    }

    const byPattern = new Map(rows.map((row) => [row.pattern, row]))
    for (const [i, entry] of DEFAULT_READY_URL_PATTERNS.entries()) {
      const legacy = byPattern.get(entry.pattern)
      if (legacy && !legacy.key) {
        db.update(readyUrlPatterns)
          .set({
            key: entry.key,
            label: entry.label,
            flags: entry.flags,
            sort_order: i,
            updated_at: sql`(datetime('now'))`,
          })
          .where(eq(readyUrlPatterns.id, legacy.id))
          .run()
      }
    }

    const existingKeys = new Set(
      this.list()
        .map((row) => row.key)
        .filter((key): key is string => !!key)
    )

    const maxOrder =
      db
        .select({
          max: sql<number | null>`max(${readyUrlPatterns.sort_order})`,
        })
        .from(readyUrlPatterns)
        .get()?.max ?? -1

    let nextOrder = maxOrder + 1
    for (const entry of DEFAULT_READY_URL_PATTERNS) {
      if (existingKeys.has(entry.key)) continue
      db.insert(readyUrlPatterns)
        .values({
          key: entry.key,
          label: entry.label,
          pattern: entry.pattern,
          flags: entry.flags,
          sort_order: nextOrder++,
        })
        .run()
    }
  },

  create(input: {
    label: string
    pattern: string
    flags?: string
  }): ReadyUrlPatternRow {
    const maxOrder =
      db
        .select({
          max: sql<number | null>`max(${readyUrlPatterns.sort_order})`,
        })
        .from(readyUrlPatterns)
        .get()?.max ?? -1

    return db
      .insert(readyUrlPatterns)
      .values({
        key: null,
        label: input.label,
        pattern: input.pattern,
        flags: input.flags ?? "i",
        sort_order: maxOrder + 1,
      })
      .returning()
      .get()
  },

  update(
    id: number,
    input: { label?: string; pattern?: string; flags?: string }
  ): ReadyUrlPatternRow | null {
    if (!this.get(id)) return null
    return db
      .update(readyUrlPatterns)
      .set({
        ...(input.label !== undefined ? { label: input.label } : {}),
        ...(input.pattern !== undefined ? { pattern: input.pattern } : {}),
        ...(input.flags !== undefined ? { flags: input.flags } : {}),
        updated_at: sql`(datetime('now'))`,
      })
      .where(eq(readyUrlPatterns.id, id))
      .returning()
      .get()
  },

  delete(id: number): boolean {
    return (
      db.delete(readyUrlPatterns).where(eq(readyUrlPatterns.id, id)).run()
        .changes > 0
    )
  },
}
