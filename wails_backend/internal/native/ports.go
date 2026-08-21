package native

import (
	"fmt"
	"strconv"
	"strings"
)

const UserPortMin = 1024
const UserPortMax = 49151

type ListeningProcess struct {
	Port int64
	PID  int64
	Name string
}

func parseLocalPort(address string) (int64, bool) {
	i := strings.LastIndex(address, ":")
	if i < 0 {
		return 0, false
	}
	n, err := strconv.ParseInt(address[i+1:], 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

func isUserPort(port int64) bool {
	return port >= UserPortMin && port <= UserPortMax
}

func finalizeEntries(entries []ListeningProcess) []ListeningProcess {
	seen := map[string]struct{}{}
	out := make([]ListeningProcess, 0, len(entries))
	for _, e := range entries {
		if !isUserPort(e.Port) || e.PID <= 0 {
			continue
		}
		key := fmt.Sprintf("%d:%d", e.Port, e.PID)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, e)
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Port < out[i].Port || (out[j].Port == out[i].Port && out[j].PID < out[i].PID) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func windowsProcessNames(pids []int64) map[int64]string {
	wanted := map[int64]struct{}{}
	for _, p := range pids {
		wanted[p] = struct{}{}
	}
	names := map[int64]string{}
	if len(wanted) == 0 {
		return names
	}
	res, err := Run([]string{"tasklist", "/FO", "CSV", "/NH"})
	if err != nil || (res.Code != 0 && strings.TrimSpace(res.Stdout) == "") {
		return names
	}
	for _, line := range strings.Split(res.Stdout, "\n") {
		line = strings.TrimRight(line, "\r")
		if !strings.HasPrefix(line, `"`) {
			continue
		}
		parts := strings.SplitN(line, `","`, 3)
		if len(parts) < 2 {
			continue
		}
		name := strings.TrimPrefix(parts[0], `"`)
		pidStr := strings.Trim(parts[1], `"`)
		pid, err := strconv.ParseInt(pidStr, 10, 64)
		if err != nil {
			continue
		}
		if _, ok := wanted[pid]; !ok {
			continue
		}
		names[pid] = name
	}
	return names
}

func listListeningProcessesWin() ([]ListeningProcess, error) {
	res, err := Run([]string{"netstat", "-ano", "-p", "TCP"})
	if err != nil {
		return nil, err
	}
	var raw []ListeningProcess
	for _, line := range strings.Split(res.Stdout, "\n") {
		parts := strings.Fields(strings.TrimSpace(strings.TrimRight(line, "\r")))
		if len(parts) < 5 || parts[0] != "TCP" {
			continue
		}
		if parts[3] != "LISTENING" {
			continue
		}
		port, ok := parseLocalPort(parts[1])
		if !ok {
			continue
		}
		pid, err := strconv.ParseInt(parts[4], 10, 64)
		if err != nil {
			continue
		}
		raw = append(raw, ListeningProcess{Port: port, PID: pid})
	}
	pids := make([]int64, len(raw))
	for i, r := range raw {
		pids[i] = r.PID
	}
	names := windowsProcessNames(pids)
	for i := range raw {
		name := names[raw[i].PID]
		if name == "" {
			name = "Unknown"
		}
		raw[i].Name = name
	}
	return finalizeEntries(raw), nil
}

func listListeningProcessesUnix() ([]ListeningProcess, error) {
	lsof, _ := Run([]string{"lsof", "-nP", "-iTCP", "-sTCP:LISTEN"})
	if lsof.Code == 0 || strings.TrimSpace(lsof.Stdout) != "" {
		var entries []ListeningProcess
		lines := strings.Split(lsof.Stdout, "\n")
		if len(lines) > 1 {
			for _, line := range lines[1:] {
				parts := strings.Fields(strings.TrimSpace(strings.TrimRight(line, "\r")))
				if len(parts) < 9 {
					continue
				}
				name := parts[0]
				pid, err := strconv.ParseInt(parts[1], 10, 64)
				if err != nil {
					continue
				}
				addr := strings.TrimSpace(parts[len(parts)-1])
				if i := strings.Index(addr, "("); i >= 0 {
					addr = strings.TrimSpace(addr[:i])
				}
				port, ok := parseLocalPort(addr)
				if !ok {
					continue
				}
				entries = append(entries, ListeningProcess{Port: port, PID: pid, Name: name})
			}
		}
		if len(entries) > 0 || lsof.Code == 0 {
			return finalizeEntries(entries), nil
		}
	}

	ss, _ := Run([]string{"ss", "-lntp"})
	if ss.Code == 0 || strings.TrimSpace(ss.Stdout) != "" {
		var entries []ListeningProcess
		for _, line := range strings.Split(ss.Stdout, "\n") {
			line = strings.TrimSpace(strings.TrimRight(line, "\r"))
			parts := strings.Fields(line)
			if len(parts) == 0 {
				continue
			}
			if !strings.HasPrefix(parts[0], "LISTEN") && parts[0] != "tcp" {
				continue
			}
			local := ""
			for _, p := range parts {
				if strings.Contains(p, ":") && parsePortOK(p) {
					local = p
					break
				}
			}
			if local == "" && len(parts) > 3 {
				local = parts[3]
			}
			port, ok := parseLocalPort(local)
			if !ok {
				continue
			}
			name := "Unknown"
			pid := int64(0)
			if i := strings.Index(line, `users:(("`); i >= 0 {
				rest := line[i+len(`users:(("`):]
				if j := strings.Index(rest, `"`); j >= 0 {
					name = rest[:j]
				}
				if k := strings.Index(rest, "pid="); k >= 0 {
					n := rest[k+4:]
					end := 0
					for end < len(n) && n[end] >= '0' && n[end] <= '9' {
						end++
					}
					pid, _ = strconv.ParseInt(n[:end], 10, 64)
				}
			}
			if pid <= 0 {
				continue
			}
			entries = append(entries, ListeningProcess{Port: port, PID: pid, Name: name})
		}
		if len(entries) > 0 || ss.Code == 0 {
			return finalizeEntries(entries), nil
		}
	}

	return nil, fmt.Errorf("Could not list listening processes (need netstat on Windows, or lsof/ss on Unix)")
}

func parsePortOK(p string) bool {
	_, ok := parseLocalPort(p)
	return ok
}

func ListListeningProcesses() ([]ListeningProcess, error) {
	if isWindows() {
		return listListeningProcessesWin()
	}
	return listListeningProcessesUnix()
}
