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
	Finding *struct {
		OSV string `json:"osv"`
	} `json:"finding"`
}

func main() {
	allowFlag := flag.String("allow", "", "comma-separated govulncheck OSV IDs allowed by policy")
	flag.Parse()

	allowed := map[string]bool{}
	for _, id := range strings.Split(*allowFlag, ",") {
		id = strings.TrimSpace(id)
		if id != "" {
			allowed[id] = true
		}
	}

	counts := map[string]int{}
	decoder := json.NewDecoder(os.Stdin)
	for {
		var msg govulncheckMessage
		err := decoder.Decode(&msg)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to parse govulncheck JSON: %v\n", err)
			os.Exit(2)
		}
		if msg.Finding != nil && msg.Finding.OSV != "" {
			counts[msg.Finding.OSV]++
		}
	}

	if len(counts) == 0 {
		fmt.Fprintln(os.Stderr, "govulncheck: no reachable vulnerabilities found")
		return
	}

	ids := make([]string, 0, len(counts))
	for id := range counts {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var blocked []string
	for _, id := range ids {
		if allowed[id] {
			fmt.Fprintf(os.Stderr, "govulncheck: allowed %s (%d reachable trace(s))\n", id, counts[id])
			continue
		}
		blocked = append(blocked, id)
		fmt.Fprintf(os.Stderr, "govulncheck: blocked %s (%d reachable trace(s))\n", id, counts[id])
	}

	if len(blocked) > 0 {
		fmt.Fprintf(os.Stderr, "govulncheck: refusing %d unallowed vulnerability ID(s): %s\n", len(blocked), strings.Join(blocked, ", "))
		os.Exit(1)
	}
}
