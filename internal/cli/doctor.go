package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/RakhaYandra/ookami/internal/config"
	"github.com/RakhaYandra/ookami/internal/doctor"
	"github.com/RakhaYandra/ookami/internal/output"
	"github.com/RakhaYandra/ookami/internal/runner"
	"github.com/RakhaYandra/ookami/internal/scoring"
	"github.com/spf13/cobra"
)

func NewDoctorCmd() *cobra.Command {
	var fix, jsonOut, quiet bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "full health check",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg := config.Default()
			if err := cfg.Validate(); err != nil {
				return err
			}
			r := runner.NewOSRunner(5 * time.Second)
			checks := doctor.DefaultChecks(cfg, r)
			ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
			defer cancel()
			rs := doctor.RunAll(ctx, checks)
			score, status := scoring.Score(rs)
			if fix {
				fmt.Fprintln(cmd.ErrOrStderr(), "info: fix engine lands in Phase 7")
			}
			out := cmd.OutOrStdout()
			var rerr error
			switch {
			case jsonOut:
				rerr = output.RenderJSON(out, rs, score, status)
			case quiet:
				rerr = output.RenderQuiet(out, rs)
			default:
				rerr = output.RenderHuman(out, rs, score, status)
			}
			if rerr != nil {
				return rerr
			}
			if code := output.ExitCode(rs, false, false); code != 0 {
				return &ExitError{Code: code}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&fix, "fix", false, "attempt safe fixes")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output JSON")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "suppress non-essential output")
	return cmd
}
