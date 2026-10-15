// Command attacksuite runs the GarudaShield v3 release-gate attack battery
// against a LIVE engine and exits non-zero if any armed category fails.
//
// Usage:
//
//	go run ./cmd/attacksuite -url http://localhost:8080 -key <admin-secret> -config garudashield.example.json
//	gs-attacksuite -url http://localhost:8080          # loopback / no key
//
// It is intended to run in CI on every build and as a daily self-test. With
// -config it reads the engine's own config so a category whose defense is
// disabled is reported as SKIP (not a false failure) — and an unknown toggle
// name is a hard error (a config rename must not silently disarm the gate).
// A benign baseline is checked first: if it fails, the harness aborts with exit
// 2 because the environment is invalid (the attack results would be meaningless).
// Each category is reset between phases via POST /api/reset so reputation/
// rate-limit state from one category does not poison the next.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/ahkurdev/garudashield/internal/attacksuite"
	"github.com/ahkurdev/garudashield/internal/config"
)

func verdictMatches(v attacksuite.Verdict, status int) bool {
	if v == attacksuite.Block {
		return status >= 400
	}
	return status < 400
}

func reset(client *http.Client, base, key string) {
	req, err := http.NewRequest(http.MethodPost, base+"/api/reset", nil)
	if err != nil {
		return
	}
	if key != "" {
		req.Header.Set("x-gs-key", key)
	}
	if resp, derr := client.Do(req); derr == nil {
		_ = resp.Body.Close()
	}
}

func main() {
	base := flag.String("url", "http://localhost:8080", "engine base URL")
	key := flag.String("key", "", "admin secret (x-gs-key); omit for loopback")
	cfgPath := flag.String("config", "", "engine config file; a category whose defense is disabled is reported SKIP, not failed")
	flag.Parse()

	client := &http.Client{Timeout: 5 * time.Second}

	// Load the engine config (optional). It lets a disabled defense be skipped
	// instead of reported as a false failure, and catches toggle renames.
	var cfg *config.Config
	if *cfgPath != "" {
		loaded, err := config.Load(*cfgPath)
		if err != nil {
			fmt.Printf("FATAL: cannot read config %s: %v\n", *cfgPath, err)
			os.Exit(2)
		}
		cfg = &loaded
	}

	// Reset engine state before the battery so prior runs don't poison this one.
	reset(client, *base, *key)

	// Baseline: benign controls MUST pass. If they don't, the environment is
	// invalid (bad UA, access gate on, engine down) and every "block" would be
	// meaningless — abort with a distinct exit code instead of a false green.
	baselineOK := true
	for _, c := range attacksuite.Baseline() {
		status, err := c.RoundTrip(nil, *base)
		if err != nil {
			fmt.Printf("[baseline] %s ERROR %v\n", c.Name, err)
			baselineOK = false
			continue
		}
		if !verdictMatches(c.Verdict, status) {
			fmt.Printf("[baseline] %s -> %d (want %s)\n", c.Name, status, c.Verdict)
			baselineOK = false
		}
	}
	if !baselineOK {
		fmt.Println("\nBASELINE FAILED: engine unreachable, access gate on, or tool UA blocked — attack results would be invalid")
		os.Exit(2)
	}

	fail, skipped, total := 0, 0, 0
	for _, cat := range attacksuite.Categories() {
		if cfg != nil {
			armed, known := cat.Armed(*cfg)
			if !known {
				fmt.Printf("[%s] ERROR unknown dependency %q\n", cat.ID, cat.Depends)
				fail++
				continue
			}
			if !armed {
				fmt.Printf("--- %s: SKIP (defense disabled: %s)\n", cat.ID, cat.Depends)
				skipped++
				continue
			}
		}
		catFail := 0
		for _, c := range cat.Cases {
			total++
			status, err := c.RoundTrip(nil, *base)
			if err != nil {
				fmt.Printf("[%s] %s ERROR %v\n", cat.ID, c.Name, err)
				catFail++
				fail++
				continue
			}
			ok := verdictMatches(c.Verdict, status)
			mark := "ok"
			if !ok {
				mark = "FAIL"
				catFail++
				fail++
			}
			fmt.Printf("[%s] %s %s -> %d (%s)\n", cat.ID, c.Name, mark, status, c.Verdict)
		}
		fmt.Printf("--- %s: %s (%d cases)\n", cat.ID, cat.Title, len(cat.Cases))
		if catFail > 0 {
			fmt.Printf("    ^^^ CATEGORY %s FAILED (%d/%d)\n", cat.ID, catFail, len(cat.Cases))
		}
		// Reset between categories so reputation/rate-limit state doesn't carry over.
		reset(client, *base, *key)
	}

	fmt.Printf("\nSUMMARY: %d/%d cases passed, %d skipped, %d failed\n", total-fail, total, skipped, fail)
	if fail > 0 {
		os.Exit(1)
	}
}
