# Workspace Manager: a guide for the agent

Workspace Manager is a desktop app that organizes a developer's projects. This
server gives you its data and tells you what to run. **It runs nothing itself**:
you run the commands, in your own shell. It also has no git or AI features;
those belong to the user, in the app.

## The model

- **Workspace**: a group of apps.
- **App**: one project folder (`project_path`).
- **Config set**: a named configuration of an app (for example `local`,
  `staging`). One is active. It holds:
  - **env vars**: `KEY=value` pairs;
  - **templates**: files in the project (such as `.env`) rendered with the env
    vars (`{{KEY}}`) before a run and restored after it;
  - four **command lists**, each with a mode (`parallel` or `sequential`):
    `setup` (one-time preparation), `dev` (start for development), `build`,
    `prod` (start in production).
- **Blueprint**: a recipe of commands that scaffolds a new app.

Start with `wm_overview`. Apps, workspaces, config sets and blueprints accept
an id or a name.

## Reading and changing configuration

You can read and edit everything about the model: `wm_app`, `wm_config`
(config sets and command lists), `wm_env`, `wm_template`, `wm_blueprint`,
`wm_system` (settings, reading project files). The app notices your edits and
refreshes by itself.

- Prefer reading before writing: `wm_app get` shows the app, its config sets and
  the active set in full.
- When you save a command list, send the whole list; it replaces the old one.
- Keep template files as the user wrote them: change only what was asked.

### Values hidden from you

Env vars and templates the user marked "excluded from AI" come back with their
value blanked, and writes to them are refused as "not found". That is
intentional (they are usually secrets). Do not try to guess or recover them,
and do not put secrets you do know into command lists or templates you save.

## Running an app

Use `wm_run`; never invent your own way to start an app, because the config set
decides the commands, the environment and the files.

1. `wm_run plan` (kind `dev`, `prod`, `build` or `setup`; optional `configSet`)
   shows the commands, mode, visible env vars, the names of hidden ones and the
   templates that will be written. Check it makes sense for the task.
2. `wm_run prepare` reserves the app, writes the templates into the project and
   returns one `run` line per command.
3. Run those `run` lines in a shell, following the mode: `parallel` means start
   them all at once, each in its own shell (they are usually servers);
   `sequential` means one after another, stopping at the first failure. Do not
   copy the underlying command instead: the `run` line supplies the complete
   environment, including the hidden values.
4. When you are finished, or the work is abandoned, call `wm_run finish`. It
   restores the original files and releases the app. **Always call it**, even
   after a failure.

An app has one session at a time, shared with the Workspace Manager app: if the
user is running it there, `prepare` fails and `wm_run status` says who holds it.
Do not work around it. Check `wm_overview` or `wm_run status` before starting.

## Creating an app from a blueprint

1. `wm_blueprint list` and `show` to pick a recipe.
2. `wm_blueprint plan_app` (blueprint, workspace, parent folder, name) returns
   the resolved commands, the folder to create and whether to `git init`.
3. Run those commands yourself, in order.
4. Register the result: `wm_app create` with the workspace and `app_dir`.

## Port conflicts

`wm_ports` lists what is listening on which port. Stopping a process is the
user's call: tell them what holds the port instead.

## Etiquette

- The data is the user's. Make the change that was asked for, and say what you
  changed.
- Delete only what you were told to delete (`wm_app delete`, `wm_config
  delete`, ...).
- If a tool refuses something, report it rather than finding a way around.
