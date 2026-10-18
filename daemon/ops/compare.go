package ops

import (
	"fmt"

	"workspace-manager/shared/api/types"
)

// ConfigSetsCompare diffs two config sets of the same app (the rule copyFrom
// has, so a part can then be copied across). With redact, env vars and
// templates that either side marks include_in_ai=false are compared here but
// returned as a status only, never with their values.
func (o *Ops) ConfigSetsCompare(leftID, rightID int64, redact bool) (types.ConfigSetCompare, error) {
	left, err := o.DB.GetConfigSetT(o.Ctx(), leftID)
	if err != nil {
		return types.ConfigSetCompare{}, err
	}
	right, err := o.DB.GetConfigSetT(o.Ctx(), rightID)
	if err != nil {
		return types.ConfigSetCompare{}, err
	}
	if left.AppID != right.AppID {
		return types.ConfigSetCompare{}, fmt.Errorf("config sets must belong to the same app")
	}
	ld, err := o.DB.ConfigSetDetailT(o.Ctx(), left)
	if err != nil {
		return types.ConfigSetCompare{}, err
	}
	rd, err := o.DB.ConfigSetDetailT(o.Ctx(), right)
	if err != nil {
		return types.ConfigSetCompare{}, err
	}
	return CompareDetails(ld, rd, redact), nil
}

func strp(s string) *string { return &s }

// CompareDetails is the pure diff of two config sets.
func CompareDetails(l, r types.ConfigSetDetail, redact bool) types.ConfigSetCompare {
	out := types.ConfigSetCompare{
		Left: l.ConfigSet, Right: r.ConfigSet,
		Env: []types.ConfigSetEnvDiff{}, Templates: []types.ConfigSetTemplateDiff{},
		Commands: []types.ConfigSetCommandListDiff{},
	}

	// Env vars by key, left order first, then right-only keys.
	rEnv := map[string]types.EnvVar{}
	for _, v := range r.EnvVars {
		rEnv[v.Key] = v
	}
	seen := map[string]bool{}
	for _, lv := range l.EnvVars {
		seen[lv.Key] = true
		d := types.ConfigSetEnvDiff{Key: lv.Key, Left: strp(lv.Value)}
		hidden := !lv.IncludeInAI
		if rv, ok := rEnv[lv.Key]; ok {
			d.Right = strp(rv.Value)
			hidden = hidden || !rv.IncludeInAI
			if lv.Value == rv.Value && lv.IncludeInAI == rv.IncludeInAI {
				d.Status = types.DiffSame
			} else {
				d.Status = types.DiffDifferent
			}
		} else {
			d.Status = types.DiffOnlyLeft
		}
		if redact && hidden {
			d.Left, d.Right, d.Redacted = nil, nil, true
		}
		out.Env = append(out.Env, d)
	}
	for _, rv := range r.EnvVars {
		if seen[rv.Key] {
			continue
		}
		d := types.ConfigSetEnvDiff{Key: rv.Key, Status: types.DiffOnlyRight, Right: strp(rv.Value)}
		if redact && !rv.IncludeInAI {
			d.Right, d.Redacted = nil, true
		}
		out.Env = append(out.Env, d)
	}

	// Templates by file path.
	rTpl := map[string]types.Template{}
	for _, t := range r.Templates {
		rTpl[t.FilePath] = t
	}
	seen = map[string]bool{}
	for _, lt := range l.Templates {
		seen[lt.FilePath] = true
		d := types.ConfigSetTemplateDiff{FilePath: lt.FilePath, Left: strp(lt.Content)}
		hidden := !lt.IncludeInAI
		if rt, ok := rTpl[lt.FilePath]; ok {
			d.Right = strp(rt.Content)
			hidden = hidden || !rt.IncludeInAI
			if lt.Content == rt.Content && lt.IncludeInAI == rt.IncludeInAI {
				d.Status = types.DiffSame
			} else {
				d.Status = types.DiffDifferent
			}
		} else {
			d.Status = types.DiffOnlyLeft
		}
		if redact && hidden {
			d.Left, d.Right, d.Redacted = nil, nil, true
		}
		out.Templates = append(out.Templates, d)
	}
	for _, rt := range r.Templates {
		if seen[rt.FilePath] {
			continue
		}
		d := types.ConfigSetTemplateDiff{FilePath: rt.FilePath, Status: types.DiffOnlyRight, Right: strp(rt.Content)}
		if redact && !rt.IncludeInAI {
			d.Right, d.Redacted = nil, true
		}
		out.Templates = append(out.Templates, d)
	}

	for _, k := range []struct {
		kind string
		l, r *types.RunConfig
	}{
		{types.KindSetup, l.SetupConfig, r.SetupConfig},
		{types.KindDev, l.DevConfig, r.DevConfig},
		{types.KindBuild, l.BuildConfig, r.BuildConfig},
		{types.KindProd, l.ProdConfig, r.ProdConfig},
	} {
		out.Commands = append(out.Commands, compareCommands(k.kind, k.l, k.r))
	}
	return out
}

func cmdsOf(c *types.RunConfig) ([]types.RunCommand, string) {
	if c == nil {
		return nil, ""
	}
	return c.Commands, c.Mode
}

type cmdSlot struct {
	label string
	n     int
}

func indexCommands(cmds []types.RunCommand) (map[cmdSlot]types.RunCommand, []cmdSlot) {
	m := map[cmdSlot]types.RunCommand{}
	var order []cmdSlot
	counts := map[string]int{}
	for _, c := range cmds {
		lb := ""
		if c.Label != nil {
			lb = *c.Label
		}
		s := cmdSlot{lb, counts[lb]}
		counts[lb]++
		m[s] = c
		order = append(order, s)
	}
	return m, order
}

// compareCommands matches commands by label (n-th occurrence to n-th
// occurrence) and unlabeled ones by position among the unlabeled.
func compareCommands(kind string, lc, rc *types.RunConfig) types.ConfigSetCommandListDiff {
	lcmds, lmode := cmdsOf(lc)
	rcmds, rmode := cmdsOf(rc)
	res := types.ConfigSetCommandListDiff{Kind: kind, LeftMode: lmode, RightMode: rmode, Commands: []types.ConfigSetCommandDiff{}}
	lm, lorder := indexCommands(lcmds)
	rm, rorder := indexCommands(rcmds)
	for _, s := range lorder {
		lcmd := lm[s]
		d := types.ConfigSetCommandDiff{Label: s.label, Left: strp(lcmd.Command)}
		if rcmd, ok := rm[s]; ok {
			d.Right = strp(rcmd.Command)
			if rcmd.Command == lcmd.Command {
				d.Status = types.DiffSame
			} else {
				d.Status = types.DiffChanged
			}
		} else {
			d.Status = types.DiffOnlyLeft
		}
		res.Commands = append(res.Commands, d)
	}
	for _, s := range rorder {
		if _, ok := lm[s]; ok {
			continue
		}
		res.Commands = append(res.Commands, types.ConfigSetCommandDiff{
			Label: s.label, Status: types.DiffOnlyRight, Right: strp(rm[s].Command),
		})
	}
	res.Status = types.DiffSame
	if lmode != rmode {
		res.Status = types.DiffDifferent
	}
	for _, c := range res.Commands {
		if c.Status != types.DiffSame {
			res.Status = types.DiffDifferent
		}
	}
	return res
}
