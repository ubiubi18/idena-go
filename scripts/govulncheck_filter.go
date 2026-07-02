package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

type govulncheckMessage struct {
	Finding *govulncheckFinding `json:"finding"`
}

type govulncheckFinding struct {
	OSV   string             `json:"osv"`
	Trace []govulncheckFrame `json:"trace"`
}

type govulncheckFrame struct {
	Module  string `json:"module"`
	Package string `json:"package"`
}

type allowPolicy struct {
	allowAll bool
	modules  map[string]bool
}

type findingKey struct {
	osv    string
	module string
}

func main() {
	allowFlag := flag.String("allow", "", "comma-separated govulncheck OSV IDs allowed by policy; use OSV@module to scope an allowance")
	flag.Parse()

	os.Exit(runFilter(os.Stdin, os.Stderr, *allowFlag))
}

func runFilter(input io.Reader, stderr io.Writer, allowList string) int {
	allowed := parseAllowList(allowList)

	counts := map[findingKey]int{}
	decoder := json.NewDecoder(input)
	for {
		var msg govulncheckMessage
		err := decoder.Decode(&msg)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(stderr, "failed to parse govulncheck JSON: %v\n", err)
			return 2
		}
		if msg.Finding != nil && msg.Finding.OSV != "" {
			counts[findingKey{
				osv:    msg.Finding.OSV,
				module: findingModule(msg.Finding),
			}]++
		}
	}

	if len(counts) == 0 {
		fmt.Fprintln(stderr, "govulncheck: no reachable vulnerabilities found")
		return 0
	}

	keys := make([]findingKey, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].osv == keys[j].osv {
			return keys[i].module < keys[j].module
		}
		return keys[i].osv < keys[j].osv
	})

	var blocked []string
	for _, key := range keys {
		if allowed[key.osv].allowsModule(key.module) {
			fmt.Fprintf(stderr, "govulncheck: allowed %s in %s (%d reachable trace(s))\n", key.osv, displayModule(key.module), counts[key])
			continue
		}
		blocked = append(blocked, displayFinding(key))
		fmt.Fprintf(stderr, "govulncheck: blocked %s in %s (%d reachable trace(s))\n", key.osv, displayModule(key.module), counts[key])
	}

	if len(blocked) > 0 {
		fmt.Fprintf(stderr, "govulncheck: refusing %d unallowed vulnerability finding(s): %s\n", len(blocked), strings.Join(blocked, ", "))
		return 1
	}
	return 0
}

func parseAllowList(raw string) map[string]allowPolicy {
	allowed := map[string]allowPolicy{}
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		id, module, hasModule := strings.Cut(entry, "@")
		id = strings.TrimSpace(id)
		module = strings.TrimSpace(module)
		if id == "" {
			continue
		}
		policy := allowed[id]
		if policy.modules == nil {
			policy.modules = map[string]bool{}
		}
		if !hasModule || module == "" {
			policy.allowAll = true
		} else {
			policy.modules[module] = true
		}
		allowed[id] = policy
	}
	return allowed
}

func (p allowPolicy) allowsModule(module string) bool {
	if p.allowAll {
		return true
	}
	return p.modules[module]
}

func findingModule(finding *govulncheckFinding) string {
	for _, frame := range finding.Trace {
		if frame.Module != "" {
			return frame.Module
		}
	}
	for _, frame := range finding.Trace {
		if frame.Package != "" {
			return frame.Package
		}
	}
	return ""
}

func displayFinding(key findingKey) string {
	if key.module == "" {
		return key.osv
	}
	return key.osv + "@" + key.module
}

func displayModule(module string) string {
	if module == "" {
		return "<unknown module>"
	}
	return module
}
