package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/RakhaYandra/ookami/internal/config"
	"github.com/RakhaYandra/ookami/internal/doctor"
	"github.com/RakhaYandra/ookami/internal/fixer"
	"github.com/RakhaYandra/ookami/internal/model"
	"github.com/RakhaYandra/ookami/internal/output"
	"github.com/RakhaYandra/ookami/internal/runner"
	"github.com/RakhaYandra/ookami/internal/scoring"
	"github.com/spf13/cobra"
)

func isCharDevice(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}

func NewDoctorCmd() *cobra.Command {
	var fix, jsonOut, quiet bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "full health check",
		Long:  "Run full health check across all categories.\n\n--json wins over --quiet; --verbose adds per-check details to human output.\n--fix attempts safe fixes with confirmation; --fix renders human output even with --json/--quiet.",
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
			out := cmd.OutOrStdout()
			errW := cmd.ErrOrStderr()
			// Flags() includes persistent parents (verified: cobra mergePersistentFlags),
			// so --verbose from root reads here. Standalone (no parent) → err → false.
			verbose, _ := cmd.Flags().GetBool("verbose")
			forceHuman := fix && (jsonOut || quiet)
			if forceHuman {
				fmt.Fprintln(errW, "info: --fix renders human output")
			}
			render := func(results []model.Result) error {
				switch {
				case jsonOut && !forceHuman: // --json wins over --quiet
					return output.RenderJSONBreakdown(out, results)
				case quiet && !forceHuman:
					return output.RenderQuiet(out, results)
				default:
					score, status := scoring.Score(results)
					if verbose {
						return output.RenderHumanVerbose(out, results, score, status)
					}
					return output.RenderHuman(out, results, score, status)
				}
			}
			if err := render(rs); err != nil {
				return err
			}
			if fix {
				fx := fixer.Fixer{
					Runner:  r,
					Stdin:   cmd.InOrStdin(),
					Stdout:  out,
					IsTTY:   isCharDevice(cmd.InOrStdin()),
					GetEuid: os.Geteuid,
				}
				items := fx.Plan(rs)
				if len(items) == 0 {
					fmt.Fprintln(errW, "info: no fixable items.")
				} else {
					fixed, failed, skipped := fx.Apply(ctx, items)
					fmt.Fprintf(errW, "fix: %d fixed, %d failed, %d skipped.\n", fixed, failed, skipped)
				}
				ctx2, cancel2 := context.WithTimeout(cmd.Context(), 60*time.Second)
				defer cancel2()
				rs = doctor.RunAll(ctx2, checks)
				if err := render(rs); err != nil {
					return err
				}
			}
			if code := output.ExitCode(rs, false, false); code != 0 {
				return &ExitError{Code: code}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&fix, "fix", false, "attempt safe fixes with confirmation")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output JSON")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "suppress non-essential output")
	return cmd
}
