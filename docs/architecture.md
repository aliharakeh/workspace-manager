# Architecture map

Workspace Manager has one owner of persistent data and running processes: the
`workspace-manager daemon` process. Desktop, editor and agent clients reach it through the same RPC
contract. Start with the directory that owns the behavior you want to change.

## Where to look

| Responsibility | Source |
|---|---|
| Desktop startup | `main.go` |
| Desktop connection, events and bound methods | `gui/` |
| Git and worktree operations | `gui/git/` |
| React UI and host adapter | `frontend/components/`, `frontend/hooks/`, `frontend/host.ts` |
| Daemon startup and connection lifecycle | `daemon/start.go`, `daemon/` |
| RPC permissions, parameter decoding and private-value filtering | `daemon/handlers/` |
| Validation, data writes and change notifications | `daemon/ops/` |
| Running apps, terminals, templates and blueprints | `daemon/services/` |
| Persistence, upgrades and typed repository helpers | `daemon/db/` |
| Method registry, payloads and events | `shared/api/` |
| RPC transport and client startup/handshake | `shared/rpc/`, `shared/client/` |
| Operating-system integration | `shared/native/` |
| Agent tool definitions and stdio MCP protocol | `mcp/` |

## Follow a request

For a desktop data operation, follow the frontend API adapter to `gui/bind.go`,
then the daemon handler, operation/service and database helper. MCP tools and
editor clients call daemon methods directly. Only the daemon opens SQLite, after acquiring the data-folder singleton lock.
The GUI and each stdio MCP client run as separate processes from the same executable.

Operations publish `dataChanged` after writes. Runner and blueprint services
publish their own events. The desktop relays these to the frontend; editor
clients subscribe through their own connections.

## Database files

`schema.sql` and `queries.sql` are the sqlc inputs. `db.go`, `models.go` and
`queries.sql.go` are generated; their size is not a reason to split them by hand.
`schema.go` contains explicit upgrades for existing databases.

Handwritten `repo_<domain>.go` files keep row conversions next to the operations
that use them: workspaces, apps, config sets, env vars, run configs, templates,
ready URLs, settings and worktree links. `repo_config_copy.go` owns copying
between config sets. `values.go` holds the shared SQLite boolean conversion.
`blueprints.go` already groups blueprint persistence separately.

These files share the existing `db` package and `DB` receiver. Moving between
domains does not require a new interface or another persistence layer.

## MCP files

The root `main.go` dispatches modes; `mcp/main.go` and `mcp/mcp.go` handle MCP startup and the stdio protocol. `daemon.go` holds
the daemon connection, argument helpers and reference resolution. `tools.go`
defines the tool type, schema builders and the explicit tool registry.

Each `tools_<domain>.go` owns that tool's schema and handler together. Config
selection helpers live with the config tool; blueprint resolution lives with
the blueprint tool. Add a tool to `allTools` to expose it. Permissions remain
enforced by the daemon and the method registry.

## Frontend feature modules

`components/git-panel.tsx` composes the Git feature. Its implementation lives
under `components/git/`:

- `use-git-panel.ts` owns loading, actions, selections and derived state. The
  panel calls it once; child views share that same state and action results.
- `worktrees-section.tsx` and `branches-section.tsx` render their respective
  lists and controls. Their model types declare the state and actions they use.
- `git-dialogs.tsx` connects that state to the focused dialog components.
  Creation, shared links, conflict resolution and deletion each have named files.
- `date-range-filter.tsx`, `list-actions.tsx` and `worktree-links.ts` keep the
  feature's date control, list controls and shared-path reporting together.

The public `GitPanel` import and its props stay the same. Views import the
controller's model as a type, so the controller has no runtime dependency on
the views. Dialogs stay mounted across ordinary reloads to preserve their form
state and existing reset behavior.

`components/config-sets/` holds copy selection types and transformations,
the category selector and the tabbed parts selector. `config-set-switcher.tsx`
owns the config-set actions and composes those selectors.

`components/env-vars/` holds the editable variable row and the read-only shared
row. `env-vars-panel.tsx` owns fetching, filtering, imports and panel actions.
Row edits remain local to the row, so refreshing the list keeps unsaved edits.

## Keeping changes readable

Split files at a responsibility boundary: an independent dialog, a domain's
repository operations or a tool handler. Keep small helpers with their caller
unless several domains need them. Avoid creating a package merely to reduce a
file's line count. Preserve existing API signatures, events, validation and
transaction boundaries during structural changes.

See `AGENTS.md` for the detailed contracts and required generation commands.
