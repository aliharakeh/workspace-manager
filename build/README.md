# Build directory

`wails3 build` uses the root Taskfile to generate bindings, build the frontend,
build wmd and the desktop app, and embed Windows resources. Output goes to
`build/bin`. `build.ps1` also builds wm-mcp.

`config.yml` holds v3 application metadata and development watcher settings.
The Windows resource JSON, manifest, NSIS helpers, and macOS plists are concrete
v3 assets. Keep their metadata in sync when changing the application version.
The icons in `windows/` and `appicon.png` remain the app's existing icons.

On Windows, `wails3 task package` builds the app, wmd and wm-mcp and creates an
NSIS installer in `build/bin`; it requires `makensis` on PATH. The installer
includes all three executables. It downloads the WebView2 bootstrapper as needed.

Use `wails3 task dev` for the Vite development server on port 5174 and automatic
Go rebuilds. Quitting the GUI leaves the daemon and active app runs alive.
