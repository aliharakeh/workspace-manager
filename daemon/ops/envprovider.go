package ops

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"workspace-manager/daemon/db"
	"workspace-manager/shared/api/types"
)

func (o *Ops) EnvProviderGet(shared bool, id int64) (types.EnvProvider, error) {
	if shared {
		if _, err := o.DB.GetWorkspaceConfigSet(o.Ctx(), id); err != nil {
			return types.EnvProvider{}, err
		}
	} else {
		if _, err := o.DB.GetConfigSet(o.Ctx(), id); err != nil {
			return types.EnvProvider{}, err
		}
	}
	return o.DB.EnvProvider(o.Ctx(), shared, id)
}
func (o *Ops) EnvProviderSave(shared bool, id int64, p types.EnvProvider) (types.Ok, error) {
	if _, err := o.EnvProviderGet(shared, id); err != nil {
		return types.Ok{}, err
	}
	p.Command = strings.TrimSpace(p.Command)
	if p.Timeout == 0 {
		p.Timeout = 30
	}
	if p.Timeout < 1 || p.Timeout > 600 {
		return types.Ok{}, fmt.Errorf("Timeout must be between 1 and 600 seconds")
	}
	if p.Command != "" && shared && !filepath.IsAbs(p.Directory) {
		return types.Ok{}, fmt.Errorf("Shared script requires an absolute working directory")
	}
	if err := o.DB.SaveEnvProvider(o.Ctx(), shared, id, p); err != nil {
		return types.Ok{}, err
	}
	if shared {
		s, _ := o.DB.GetWorkspaceConfigSet(o.Ctx(), id)
		o.changed("sharedSet", s.WorkspaceID)
		o.changed("envVar")
	} else {
		s, _ := o.DB.GetConfigSet(o.Ctx(), id)
		o.changed("envVar", s.AppID)
	}
	return types.Ok{Ok: true}, nil
}

func (o *Ops) EnvProviderRun(shared bool, id int64) (types.Ok, error) {
	if _, err := o.EnvProviderGet(shared, id); err != nil {
		return types.Ok{}, err
	}
	if err := o.Runner.Providers.Run(o.Ctx(), shared, id); err != nil {
		return types.Ok{}, err
	}
	if err := o.syncProviderVars(shared, id, o.Runner.Providers.Values(shared, id)); err != nil {
		return types.Ok{}, err
	}
	return types.Ok{Ok: true}, nil
}

// syncProviderVars saves script output as variables of the set, so they are
// listed and (for a shared set) can be linked into apps. Existing variables get
// the new value (their private flag is kept); new ones are created private so
// generated values stay hidden from agents. Nothing is ever deleted.
func (o *Ops) syncProviderVars(shared bool, id int64, values map[string]string) error {
	keys := make([]string, 0, len(values))
	for k := range values {
		if strings.TrimSpace(k) != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	yes := true
	if shared {
		rows, err := o.DB.ListWorkspaceEnvVars(o.Ctx(), id)
		if err != nil {
			return err
		}
		byKey := map[string]db.WorkspaceEnvVar{}
		for _, r := range rows {
			byKey[r.Key] = r
		}
		for _, k := range keys {
			v := values[k]
			if cur, ok := byKey[k]; ok {
				if cur.Value != v {
					if _, err := o.SharedVarsUpdate(cur.ID, types.EnvVarUpdateInput{Value: &v}); err != nil {
						return err
					}
				}
				continue
			}
			if _, err := o.SharedVarsCreate(id, types.EnvVarCreateInput{Key: k, Value: &v, Private: &yes}); err != nil {
				return err
			}
		}
		// Script output wins over manual variables of the same name in linked
		// app config sets too: their own rows are updated with the new value.
		linked, err := o.DB.ListConfigSetIDsLinkedToSharedSet(o.Ctx(), &id)
		if err != nil {
			return err
		}
		for _, setID := range linked {
			own, err := o.DB.ListEnvVarsByConfigSet(o.Ctx(), setID)
			if err != nil {
				return err
			}
			for _, r := range own {
				v, ok := values[r.Key]
				if r.SharedVarID != nil || !ok || r.Value == v {
					continue
				}
				if _, err := o.EnvVarsUpdate(r.ID, types.EnvVarUpdateInput{Value: &v}); err != nil {
					return err
				}
			}
		}
		if s, err := o.getSharedSet(id); err == nil {
			o.changed("sharedSet", s.WorkspaceID)
		}
		return nil
	}
	set, err := o.DB.GetConfigSet(o.Ctx(), id)
	if err != nil {
		return err
	}
	rows, err := o.DB.ListEnvVarsByConfigSet(o.Ctx(), id)
	if err != nil {
		return err
	}
	byKey := map[string]db.EnvVar{}
	for _, r := range rows {
		byKey[r.Key] = r
	}
	for _, k := range keys {
		v := values[k]
		if cur, ok := byKey[k]; ok {
			// Rows mirroring a shared variable are read-only here.
			if cur.SharedVarID == nil && cur.Value != v {
				if _, err := o.EnvVarsUpdate(cur.ID, types.EnvVarUpdateInput{Value: &v}); err != nil {
					return err
				}
			}
			continue
		}
		if _, err := o.EnvVarsCreate(set.AppID, id, types.EnvVarCreateInput{Key: k, Value: &v, Private: &yes}); err != nil {
			return err
		}
	}
	return nil
}
