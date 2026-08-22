package gitgraph

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type RemoteInfo struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Web  string `json:"web,omitempty"`
	Host string `json:"host,omitempty"`
	SSH  bool   `json:"ssh"`
}

func ListRemote(path string) (RemoteInfo, error) {
	root, err := gitRoot(path)
	if err != nil {
		return RemoteInfo{}, err
	}
	return listRemote(root)
}

func FetchRemote(path string) error {
	root, err := gitRoot(path)
	if err != nil {
		return err
	}
	info, err := listRemote(root)
	if err != nil {
		return err
	}
	return fetchRemote(root, info)
}

func listRemote(root string) (RemoteInfo, error) {
	name := "origin"
	u, err := gitOutput(root, "remote", "get-url", "origin")
	if err != nil || u == "" {
		names, e := gitOutput(root, "remote")
		if e != nil || names == "" {
			return RemoteInfo{}, fmt.Errorf("no git remotes")
		}
		name = strings.Fields(names)[0]
		u, err = gitOutput(root, "remote", "get-url", name)
		if err != nil || u == "" {
			return RemoteInfo{}, fmt.Errorf("no git remotes")
		}
	}
	web := toWebBase(u)
	return RemoteInfo{Name: name, URL: u, Web: web, Host: remoteHost(web), SSH: isSSHURL(u)}, nil
}

func fetchRemote(root string, info RemoteInfo) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", root, "fetch", "--prune", info.Name)
	hideWindow(cmd)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("fetch timed out")
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func remoteHost(web string) string {
	if _, rest, ok := strings.Cut(web, "://"); ok {
		host, _, _ := strings.Cut(rest, "/")
		return host
	}
	return ""
}

func isSSHURL(u string) bool {
	u = strings.TrimSpace(u)
	return strings.HasPrefix(u, "git@") || strings.HasPrefix(u, "ssh://")
}
