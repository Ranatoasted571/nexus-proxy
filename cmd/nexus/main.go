package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/lynuxis2026-pixel/nexus-proxy/internal/config"
	"github.com/lynuxis2026-pixel/nexus-proxy/internal/dashboard"
	"github.com/lynuxis2026-pixel/nexus-proxy/internal/providers"
	"github.com/lynuxis2026-pixel/nexus-proxy/internal/proxy"
	"github.com/lynuxis2026-pixel/nexus-proxy/internal/storage"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "nexus",
	Short: "Smart proxy + dashboard for Claude Code",
	Long: `
███╗   ██╗███████╗██╗  ██╗██╗   ██╗███████╗
████╗  ██║██╔════╝╚██╗██╔╝██║   ██║██╔════╝
██╔██╗ ██║█████╗   ╚███╔╝ ██║   ██║███████╗
██║╚██╗██║██╔══╝   ██╔██╗ ██║   ██║╚════██║
██║ ╚████║███████╗██╔╝ ██╗╚██████╔╝███████║
╚═╝  ╚═══╝╚══════╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝

Route Claude Code to any LLM. Free, local, or cloud.
`,
}

// start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start proxy + dashboard",
	Long:  "Start the NEXUS proxy server and dashboard",
	RunE:  runStart,
}

// add command
var addCmd = &cobra.Command{
	Use:   "add [provider] [api-key]",
	Short: "Add a provider",
	Example: `  nexus add deepseek sk-xxx
  nexus add groq gsk-xxx
  nexus add gemini AIza-xxx
  nexus add ollama`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runAdd,
}

// status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show provider health",
	RunE:  runStatus,
}

// logs command
var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show recent requests",
	RunE:  runLogs,
}

// cost command
var costCmd = &cobra.Command{
	Use:   "cost",
	Short: "Show cost breakdown",
	RunE:  runCost,
}

// config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show or edit config",
	RunE:  runConfig,
}

// models command
var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Show how Claude models map to each provider",
	RunE:  runModels,
}

// version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("nexus %s (built %s)\n", Version, BuildTime)
	},
}

// Flags
var (
	flagProxyPort     int
	flagDashboardPort int
	flagDev           bool
	flagNoUI          bool
	flagLogLevel      string
	flagAddType       string
	flagAddBaseURL    string
	flagAddRegion     string
	flagAddProject    string
	flagAddAPIVersion string
	flagBudget        float64
)

func init() {
	startCmd.Flags().IntVarP(&flagProxyPort, "port", "p", 3000, "Proxy port")
	startCmd.Flags().IntVarP(&flagDashboardPort, "ui", "u", 2222, "Dashboard port")
	startCmd.Flags().BoolVar(&flagDev, "dev", false, "Development mode")
	startCmd.Flags().BoolVar(&flagNoUI, "no-ui", false, "Disable dashboard")
	startCmd.Flags().StringVarP(&flagLogLevel, "log", "l", "info", "Log level (debug|info|warn|error)")
	startCmd.Flags().Float64Var(&flagBudget, "budget", 0, "Daily budget cap in USD (0 = unlimited; free/local only once exceeded)")

	addCmd.Flags().StringVar(&flagAddType, "type", "", "Provider type: openai-compatible | azure | vertex | bedrock")
	addCmd.Flags().StringVar(&flagAddBaseURL, "base-url", "", "Base URL (custom/azure endpoint; optional for ollama)")
	addCmd.Flags().StringVar(&flagAddRegion, "region", "", "Region (AWS Bedrock / Google Vertex)")
	addCmd.Flags().StringVar(&flagAddProject, "project", "", "Project ID (Google Vertex)")
	addCmd.Flags().StringVar(&flagAddAPIVersion, "api-version", "", "API version (Azure OpenAI)")

	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(costCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(modelsCmd)
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		color.Red("Error: %v", err)
		os.Exit(1)
	}
}

// ─── Command implementations ───────────────────────────────────────────────

func runStart(cmd *cobra.Command, args []string) error {
	if flagDev {
		flagLogLevel = "debug"
	}
	setupLogging(flagLogLevel)

	// Shared storage + event broker (the broker feeds the dashboard live feed).
	db, err := storage.New(storage.DefaultDBPath())
	if err != nil {
		return fmt.Errorf("failed to open storage: %w", err)
	}
	defer db.Close()

	broker := dashboard.NewSSEBroker()

	cfg := &proxy.Config{
		Port:           flagProxyPort,
		DashboardPort:  flagDashboardPort,
		LogLevel:       flagLogLevel,
		DailyBudgetUSD: flagBudget,
	}

	proxySrv, err := proxy.New(cfg, db, broker)
	if err != nil {
		return fmt.Errorf("failed to create proxy server: %w", err)
	}

	// Cancel the root context on Ctrl+C / SIGTERM for a graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Dashboard server (optional). Shares the broker + DB with the proxy.
	var dash *dashboard.Server
	if !flagNoUI {
		dash = dashboard.NewServer(flagDashboardPort, broker, db)
		go func() {
			if err := dash.Start(); err != nil && err != http.ErrServerClosed {
				log.Error().Err(err).Msg("Dashboard server error")
			}
		}()
	}

	printReady(flagProxyPort, flagDashboardPort, !flagNoUI)

	// Run the proxy. Start blocks until ctx is cancelled or a fatal error occurs.
	runErr := proxySrv.Start(ctx)

	// Graceful shutdown.
	fmt.Println()
	color.Yellow("→ Shutting down NEXUS...")
	if dash != nil {
		if err := dash.Shutdown(); err != nil {
			log.Warn().Err(err).Msg("Dashboard shutdown error")
		}
	}
	color.Green("✓ Stopped")
	return runErr
}

func runAdd(cmd *cobra.Command, args []string) error {
	name := strings.ToLower(args[0])
	apiKey := ""
	if len(args) > 1 {
		apiKey = args[1]
	}

	pc := config.Provider{Name: name, APIKey: apiKey, Type: flagAddType, BaseURL: flagAddBaseURL}

	if flagAddType != "" {
		// Custom / enterprise provider (openai-compatible, azure, vertex, bedrock).
		pc.Region = flagAddRegion
		pc.Project = flagAddProject
		pc.APIVersion = flagAddAPIVersion
		if pc.Tier == "" {
			pc.Tier = "standard"
		}
		if _, err := providers.New(specFromConfig(pc)); err != nil {
			return fmt.Errorf("invalid provider config: %w", err)
		}
	} else {
		pc.Tier = providers.DefaultTier(name)
		pc.Models = providers.DefaultModels(name)
		if name == "ollama" && pc.BaseURL == "" {
			pc.BaseURL = "http://localhost:11434"
		}
		if _, err := providers.FromConfig(name, apiKey, pc.BaseURL, pc.Models); err != nil {
			return fmt.Errorf("unknown provider %q — for a custom endpoint use --type (openai-compatible|azure|vertex|bedrock)", name)
		}
	}

	path := config.DefaultPath()
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	cfg.Upsert(pc)
	if err := config.Save(path, cfg); err != nil {
		return err
	}

	color.Green("✓ Added provider: %s (tier: %s)", name, pc.Tier)
	color.White("  Config: %s", path)
	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(config.DefaultPath())
	if err != nil {
		return err
	}
	if len(cfg.Providers) == 0 {
		color.Yellow("No providers configured. Add one, e.g.:  nexus add groq <key>")
		return nil
	}

	color.Cyan("Provider health:\n")
	for _, pc := range cfg.Providers {
		impl, err := providers.New(specFromConfig(pc))
		if err != nil {
			color.Red("  ✗ %-11s %v", pc.Name, err)
			continue
		}
		if herr := impl.HealthCheck(); herr != nil {
			color.Red("  ● %-11s [%-8s] DOWN  (%v)", impl.Name(), impl.Tier(), herr)
		} else {
			color.Green("  ● %-11s [%-8s] OK", impl.Name(), impl.Tier())
		}
	}
	return nil
}

// runModels shows how each provider maps the three Claude tiers.
func runModels(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(config.DefaultPath())
	if err != nil {
		return err
	}
	if len(cfg.Providers) == 0 {
		color.Yellow("No providers configured. Add one, e.g.:  nexus add groq <key>")
		return nil
	}
	color.Cyan("Model mapping (Claude model → provider model):\n")
	for _, pc := range cfg.Providers {
		impl, err := providers.New(specFromConfig(pc))
		if err != nil {
			continue
		}
		color.White("  %-11s [%s]", impl.Name(), impl.Tier())
		fmt.Printf("      haiku  → %s\n", impl.MapModel("claude-haiku-4-5"))
		fmt.Printf("      sonnet → %s\n", impl.MapModel("claude-sonnet-4-6"))
		fmt.Printf("      opus   → %s\n", impl.MapModel("claude-opus-4-5"))
	}
	return nil
}

// specFromConfig builds a providers.Spec from a config entry, resolving env keys.
func specFromConfig(pc config.Provider) providers.Spec {
	return providers.Spec{
		Name:        pc.Name,
		Type:        pc.Type,
		APIKey:      config.ResolveKey(pc.APIKey),
		BaseURL:     pc.BaseURL,
		Models:      pc.Models,
		Tier:        pc.Tier,
		ModelMap:    pc.ModelMap,
		InputPer1M:  pc.InputPer1M,
		OutputPer1M: pc.OutputPer1M,
		Region:      pc.Region,
		Project:     pc.Project,
		APIVersion:  pc.APIVersion,
	}
}

func runLogs(cmd *cobra.Command, args []string) error {
	db, err := storage.New(storage.DefaultDBPath())
	if err != nil {
		return err
	}
	defer db.Close()

	reqs, err := db.GetRecentRequests(20)
	if err != nil {
		return err
	}
	if len(reqs) == 0 {
		color.Yellow("No requests logged yet.")
		return nil
	}

	color.Cyan("Recent requests:\n")
	for _, q := range reqs {
		fmt.Printf("  %s  %-9s  %-8s  %-18s  %d→%d tok  $%.5f  %dms  [%d]\n",
			q.CreatedAt.Local().Format("15:04:05"), q.Provider, q.Complexity, q.ModelAsked,
			q.InputTokens, q.OutputTokens, q.CostUSD, q.LatencyMS, q.Status)
	}
	return nil
}

func runCost(cmd *cobra.Command, args []string) error {
	db, err := storage.New(storage.DefaultDBPath())
	if err != nil {
		return err
	}
	defer db.Close()

	today, _ := db.GetStats("today")
	week, _ := db.GetStats("week")
	forecast, _ := db.GetCostForecast()

	color.Cyan("Cost breakdown:\n")
	if today != nil {
		color.White("  Today:    $%.4f  (%d requests, %d tokens)",
			today.TotalCostUSD, today.TotalRequests, today.TotalInputTokens+today.TotalOutputTokens)
	}
	if week != nil {
		color.White("  Last 7d:  $%.4f  (%d requests)", week.TotalCostUSD, week.TotalRequests)
	}
	color.White("  Forecast: $%.2f / month", forecast)

	if bd, _ := db.GetProviderBreakdown(); len(bd) > 0 {
		color.Cyan("\n  By provider:")
		for _, p := range bd {
			color.White("    %-10s  $%.4f  (%d req)", p.Provider, p.TotalCostUSD, p.Requests)
		}
	}
	return nil
}

func runConfig(cmd *cobra.Command, args []string) error {
	path := config.DefaultPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		color.Yellow("No config yet at %s", path)
		color.White("  Create one by adding a provider:  nexus add groq <key>")
		return nil
	}
	color.Cyan("Config: %s", path)
	return nil
}

// ─── Helpers ───────────────────────────────────────────────────────────────

// setupLogging configures the global zerolog logger with a pretty console writer.
func setupLogging(level string) {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "15:04:05"})
}

// printReady prints the human-friendly "running" banner with connection hints.
func printReady(proxyPort, dashPort int, ui bool) {
	color.Green("\n✓ NEXUS running")
	color.White("  Proxy:     http://localhost:%d", proxyPort)
	if ui {
		color.White("  Dashboard: http://localhost:%d", dashPort)
	}
	color.Yellow("\n  Connect Claude Code:")
	color.White("  export ANTHROPIC_BASE_URL=http://localhost:%d", proxyPort)
	color.White("  export ANTHROPIC_API_KEY=nexus-local")

	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		color.HiBlack("\n  Tip: set ANTHROPIC_API_KEY in this shell so NEXUS can reach Anthropic.")
	}
	fmt.Println()
}
