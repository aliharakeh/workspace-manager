# Dynamic environment scripts

App config sets and workspace shared config sets each support one optional script in the **Configure script** dialog in their environment editor. The script prints one JSON object of strings to stdout. You do not need to declare output keys or create variables first:

```json
{"API_TOKEN":"fresh-token","API_URL":"https://api.example.test"}
```

Scripts run only when you click **Run script now**, using the saved configuration. App starts and reloads never execute a script. Run the workspace shared script first, then any app script that depends on its values. An app script receives saved variables plus the latest shared script output from its linked shared set. Refreshing one script never executes another.

All commands and templates in an app run receive the latest successful results. All output keys are merged into the runtime environment, including new names. Shared script results are reused by every app linked to that shared set. Shared output overrides matching saved values; app output is merged last and overrides matching runtime keys. A successful **Run script now** also saves each output key as a variable of that config set (new ones are created private; existing ones get the new value and keep their private flag; nothing is deleted), so they are listed and, for a shared set, can be linked into apps like any shared variable. Script output takes priority over manually entered variables of the same name: their values are overwritten, including an app's own variables in config sets linked to a shared set that ran the script. Refreshing scripts does not change the environment or templates of an already-running app. Start or reload that app to use the refreshed values.

Shared scripts require an absolute working directory. App scripts run in the app’s configured project folder; subsequent worktree runs reuse the result. Scripts use cmd.exe on Windows and sh on other platforms, without interactive input. The default timeout is 30 seconds, configurable from 1 to 600 seconds. Output is limited to 1 MiB.

A failure, timeout or invalid output leaves the last successful result unchanged. If no successful result exists, an app start fails before templates or commands, asking you to run the script manually. Changing the script configuration also requires a new manual run. Provider output is captured separately from terminal logs; errors report no raw output. Raw script output stays in memory; only the saved variables persist. Template target files and application output may contain them. Results are cleared on daemon shutdown or restart. Run scripts manually again after a daemon restart. Scheduling is not supported.

Provider configuration is edited in the GUI or, for agents, through the MCP tool `wm_env_script` (get, set, clear, run); an agent never receives generated values. Trusted exports that explicitly include secrets carry it in bundle version 3; other exports omit it. Generated results are never exported. Full environment copies carry the provider; partial copies disable it. Editing, copying, export and import never execute providers. Keep credentials out of commands; use private input variables or local credential tools.

Only the JSON structure is checked: output must be one object with string values, without trailing content. Empty objects are accepted, and duplicate JSON keys use the last value. Variable names are not checked against a naming pattern or existing variables. Clear the command and save to disable a script.
