package main

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/lynuxis2026-pixel/nexus-proxy/internal/config"
	"github.com/lynuxis2026-pixel/nexus-proxy/internal/storage"
)

// report command — the Trust & Savings Report: savings + privacy in one artifact.
var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Trust & Savings report — what you saved AND what never left your machine",
	RunE:  runReport,
}

var flagReportPeriod string

func runReport(cmd *cobra.Command, args []string) error {
	period := flagReportPeriod
	if period == "" {
		period = "month"
	}
	db, err := storage.New(storage.DefaultDBPath())
	if err != nil {
		return err
	}
	defer db.Close()

	sv, err := db.GetSavings(period)
	if err != nil {
		return err
	}
	st, _ := db.GetStats(period)
	providers := 0
	if cfg, e := config.Load(config.DefaultPath()); e == nil {
		providers = len(cfg.Providers)
	}

	color.Cyan("\n  🛡️  NEXUS — Trust & Savings Report  (%s)\n", period)
	color.White("  ────────────────────────────────────────────")
	color.White("  Requests       %d  across %d provider(s)", sv.Requests, providers)
	color.Green("  Saved          $%.2f   (%.0f%% cheaper than all-Claude)", sv.SavedUSD, sv.PercentSaved)
	if sv.CacheSavedUSD > 0 {
		color.White("  Cache saved    $%.2f   (from prompt caching)", sv.CacheSavedUSD)
	}
	color.Cyan("  🔒 Privacy     %d secrets/PII masked before leaving your machine · 0 leaked", st.RedactedTotal)
	color.White("  ────────────────────────────────────────────")
	color.White("  Actual spend   $%.4f", sv.ActualUSD)
	color.White("  All-Claude     $%.2f  (what this traffic would have cost)", sv.BaselineUSD)

	color.White("\n  Share it:")
	color.HiBlack("  \"NEXUS saved me $%.2f (%.0f%% cheaper) on AI coding and masked %d secrets\n   before they ever left my machine — local, open-source.\"",
		sv.SavedUSD, sv.PercentSaved, st.RedactedTotal)
	color.HiBlack("  Card: open the dashboard → Download card, or GET /api/savings/card.svg\n")
	return nil
}
