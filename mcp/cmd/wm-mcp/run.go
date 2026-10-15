package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"workspace-manager/api"
	"workspace-manager/api/types"
	"workspace-manager/core/db"
	"workspace-manager/core/native"
	"workspace-manager/core/services"
)

// ---- wm_run ----
//
// This server never starts a process. wm_run hands the agent what it needs to
// run an app itself: plan describes the run, prepare takes the app's session
// and writes the templates, finish puts the original files back.

var runKinds = []string{"dev", "prod", "build", "setup"}

func runTool() mcpTool {
	return mcpTool{
		Name: "wm_run",
		Description: "How to run an app. This server runs nothing: plan describes a dev/prod/build/setup run, prepare reserves the app and writes its template files and returns the commands for you to run, finish restores the files and releases the app, status says who holds it. " +
			"An app has one session at a time, shared with the Workspace Manager app.",
		Schema: schema("Run an app yourself", map[string]any{
			"action":    map[string]any{"type": "string", "description": "plan, prepare, finish, status", "enum": []string{"plan", "prepare", "finish", "status"}},
			"app":       strProp(appRefDesc),
			"kind":      map[string]any{"type": "string", "description": "dev, prod, build or setup (plan, prepare; default dev)", "enum": runKinds},
			"configSet": strProp(setRefDesc + " (plan, prepare)"),
		}, "action"),
		Handler: func(args map[string]any) (any, error) {
			app, err := resolveApp(argStr(args, "app"))
			if err != nil {
				return nil, err
			}
			appID := appIDOf(app)
			switch argStr(args, "action") {
			case "plan":
				p, err := buildRunPlan(app, args)
				if err != nil {
					return nil, err
				}
				p.Session, _ = sessionOf(appID)
				return p, nil
			case "prepare":
				return prepareRun(app, args)
			case "finish":
				return finishRun(appID)
			case "status":
				return sessionOf(appID)
			}
			return nil, fmt.Errorf("unknown action %q", argStr(args, "action"))
		},
	}
}

type planCommand struct {
	ID      int64  `json:"id"`
	Label   string `json:"label,omitempty"`
	Command string `json:"command"`
}

type planEnv struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// runPlan is everything an agent needs to know about one run, with the values
// excluded from AI left out: those are named, never shown.
type runPlan struct {
	App       string         `json:"app"`
	AppID     int64          `json:"appId"`
	Kind      string         `json:"kind"`
	ConfigSet string         `json:"configSet"`
	ConfigID  int64          `json:"configSetId"`
	Dir       string         `json:"dir"`
	Mode      string         `json:"mode"`
	Commands  []planCommand  `json:"commands"`
	Env       []planEnv      `json:"env"`
	HiddenEnv []string       `json:"hiddenEnvKeys"`
	Templates []string       `json:"templates"`
	Session   map[string]any `json:"session,omitempty"`
	Notes     []string       `json:"notes"`
}

func checkRunKind(kind string) error {
	for _, k := range runKinds {
		if k == kind {
			return nil
		}
	}
	return fmt.Errorf("kind must be one of %v, got %q", runKinds, kind)
}

func buildRunPlan(app map[string]any, args map[string]any) (*runPlan, error) {
	kind := argStr(args, "kind")
	if kind == "" {
		kind = "dev"
	}
	if err := checkRunKind(kind); err != nil {
		return nil, err
	}
	set, err := resolveConfigSet(app, argStr(args, "configSet"))
	if err != nil {
		return nil, err
	}
	var detail types.ConfigSetDetail
	if err := local.Call(api.MethodConfigSetsDetail, map[string]any{"id": set.ID}, &detail); err != nil {
		return nil, err
	}
	var cfg *types.RunConfig
	switch kind {
	case "dev":
		cfg = detail.DevConfig
	case "prod":
		cfg = detail.ProdConfig
	case "build":
		cfg = detail.BuildConfig
	case "setup":
		cfg = detail.SetupConfig
	}
	if cfg == nil || len(cfg.Commands) == 0 {
		return nil, api.Errf(api.KindNotFound, "config set %s has no %s commands", set.Name, kind)
	}
	name, _ := app["name"].(string)
	dir, _ := app["project_path"].(string)
	p := &runPlan{
		App: name, AppID: appIDOf(app), Kind: kind, ConfigSet: set.Name, ConfigID: set.ID,
		Dir: dir, Mode: cfg.Mode, Commands: []planCommand{}, Env: []planEnv{},
		HiddenEnv: []string{}, Templates: []string{},
	}
	for _, c := range cfg.Commands {
		pc := planCommand{ID: c.ID, Command: c.Command}
		if c.Label != nil {
			pc.Label = *c.Label
		}
		p.Commands = append(p.Commands, pc)
	}
	for _, v := range detail.EnvVars {
		if !v.IncludeInAI {
			p.HiddenEnv = append(p.HiddenEnv, v.Key)
			continue
		}
		p.Env = append(p.Env, planEnv{Key: v.Key, Value: v.Value})
	}
	for _, t := range detail.Templates {
		p.Templates = append(p.Templates, t.FilePath)
	}
	p.Notes = []string{
		"mode " + cfg.Mode + ": " + map[string]string{
			"parallel":   "start every command at once, each in its own shell (they usually are servers).",
			"sequential": "run the commands one after another and stop at the first failure.",
		}[cfg.Mode],
		"Run each command through the exec line prepare returns, not directly: it adds the full environment (including values hidden from you) and keeps secrets out of this conversation.",
	}
	if len(p.Templates) > 0 {
		p.Notes = append(p.Notes, "prepare renders the templates above into the app folder (backing up the originals); finish restores them. Always call finish.")
	}
	return p, nil
}

// prepareRun takes the app's session, writes its templates and returns the
// commands to run.
func prepareRun(app map[string]any, args map[string]any) (any, error) {
	p, err := buildRunPlan(app, args)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	sessionID := fmt.Sprintf("%d-%d-agent", p.AppID, nowMillis())
	claim := db.SessionClaim{
		AppID: p.AppID, SessionID: sessionID, Owner: services.OwnerAgent, Kind: p.Kind,
		PID: os.Getpid(), ConfigSetID: p.ConfigID, Dir: p.Dir,
	}
	if err := services.ClaimSession(ctx, database, claim, nil); err != nil {
		return nil, err
	}
	if err := services.ApplyTemplates(ctx, database, p.AppID, p.ConfigID, "", sessionID); err != nil {
		services.ReleaseSession(ctx, database, p.AppID, sessionID)
		return nil, fmt.Errorf("could not write the templates: %w", err)
	}
	_ = database.LogChange(ctx, "session", []int64{p.AppID})

	exe, _ := os.Executable()
	type runLine struct {
		planCommand
		Run string `json:"run"`
	}
	lines := make([]runLine, 0, len(p.Commands))
	for _, c := range p.Commands {
		lines = append(lines, runLine{c, fmt.Sprintf("%s exec --app %d --session %s --command %d",
			quoteArg(exe), p.AppID, sessionID, c.ID)})
	}
	return map[string]any{
		"sessionId": sessionID, "app": p.App, "kind": p.Kind, "mode": p.Mode, "dir": p.Dir,
		"commands": lines,
		"next":     "Run each command's \"run\" line in a shell, as the plan says (" + p.Mode + "). When you are done or stop, call wm_run finish for this app.",
	}, nil
}

// finishRun restores the templates and releases the app's session, if an
// agent holds it.
func finishRun(appID int64) (any, error) {
	ctx := context.Background()
	held, found, err := database.GetAppSession(ctx, appID)
	if err != nil {
		return nil, err
	}
	if !found {
		return map[string]any{"ok": true, "note": "the app has no session"}, nil
	}
	if held.Owner != services.OwnerAgent {
		return nil, api.Errf(api.KindConflict, "the app's session belongs to %s; only it can end it", held.Owner)
	}
	_ = services.RestoreTemplates(ctx, database, held.AppID, held.ConfigSetID, held.Dir, held.SessionID)
	services.ReleaseSession(ctx, database, appID, held.SessionID)
	_ = database.LogChange(ctx, "session", []int64{appID})
	return map[string]any{"ok": true, "note": "original files restored, session released"}, nil
}

// releaseOwnSessions ends every session this process prepared, so templates do
// not stay rendered when the agent goes away without calling finish.
func releaseOwnSessions() {
	if database == nil {
		return
	}
	ctx := context.Background()
	rows, err := database.ListAppSessions(ctx)
	if err != nil {
		return
	}
	for _, s := range rows {
		if s.Owner == services.OwnerAgent && s.PID == os.Getpid() {
			_ = services.RestoreTemplates(ctx, database, s.AppID, s.ConfigSetID, s.Dir, s.SessionID)
			services.ReleaseSession(ctx, database, s.AppID, s.SessionID)
			_ = database.LogChange(ctx, "session", []int64{s.AppID})
		}
	}
}

// ---- sessions as the agent sees them ----

func sessionMap(s db.SessionClaim) map[string]any {
	return map[string]any{
		"running": native.PidAlive(s.PID), "owner": s.Owner, "kind": s.Kind,
		"sessionId": s.SessionID, "startedAt": s.StartedAt.Format("2006-01-02 15:04:05"),
	}
}

// sessionOf describes who holds an app: running is false when nobody does, or
// when the holder's process is gone.
func sessionOf(appID int64) (map[string]any, error) {
	s, found, err := database.GetAppSession(context.Background(), appID)
	if err != nil {
		return nil, err
	}
	if !found {
		return map[string]any{"running": false}, nil
	}
	return sessionMap(s), nil
}

// sessionsView lists every app's session, by app id.
func sessionsView() (map[string]any, error) {
	rows, err := database.ListAppSessions(context.Background())
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	for _, s := range rows {
		out[strconv.FormatInt(s.AppID, 10)] = sessionMap(s)
	}
	return out, nil
}
