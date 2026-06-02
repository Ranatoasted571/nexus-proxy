package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// code command — one command to launch Claude Code through NEXUS.
var codeCmd = &cobra.Command{
	Use:                "code [-- claude args...]",
	Short:              "Launch Claude Code wired through NEXUS (auto-starts the proxy if needed)",
	DisableFlagParsing: false,
	RunE:               runCode,
}

var flagCodePort int

func proxyHealthy(base string) bool {
	c := &http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := c.Get(base + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func runCode(cmd *cobra.Command, args []string) error {
	port := flagCodePort
	if port == 0 {
		port = 3000
	}
	base := fmt.Sprintf("http://localhost:%d", port)

	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("`claude` (Claude Code) not found on PATH — install it: npm i -g @anthropic-ai/claude-code")
	}

	// Start NEXUS in the background if it isn't already reachable.
	var nexusProc *os.Process
	if proxyHealthy(base) {
		color.Green("✓ Using NEXUS already running on :%d", port)
	} else {
		self, err := os.Executable()
		if err != nil {
			return fmt.Errorf("cannot locate the nexus binary: %w", err)
		}
		start := exec.Command(self, "start", "--port", strconv.Itoa(port))
		start.Stdout = os.Stderr // keep NEXUS logs off Claude Code's stdout
		start.Stderr = os.Stderr
		if err := start.Start(); err != nil {
			return fmt.Errorf("failed to start NEXUS: %w", err)
		}
		nexusProc = start.Process
		color.Yellow("→ Starting NEXUS on :%d …", port)
		up := false
		for i := 0; i < 30; i++ {
			if proxyHealthy(base) {
				up = true
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		if !up {
			_ = nexusProc.Kill()
			return fmt.Errorf("NEXUS did not become healthy on :%d", port)
		}
		color.Green("✓ NEXUS ready on :%d", port)
	}

	// Launch Claude Code pointed at NEXUS.
	color.Cyan("→ Launching Claude Code through NEXUS\n")
	c := exec.Command(claudePath, args...)
	c.Env = append(os.Environ(),
		"ANTHROPIC_BASE_URL="+base,
		"ANTHROPIC_API_KEY=nexus-local",
	)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	runErr := c.Run()

	// Only stop NEXUS if this command started it.
	if nexusProc != nil {
		color.Yellow("\n→ Stopping NEXUS …")
		_ = nexusProc.Kill()
		_, _ = nexusProc.Wait()
	}
	return runErr
}
