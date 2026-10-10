package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/RakhaYandra/ookami/internal/config"
	"github.com/RakhaYandra/ookami/internal/doctor"
	"github.com/RakhaYandra/ookami/internal/model"
	"github.com/RakhaYandra/ookami/internal/output"
	"github.com/RakhaYandra/ookami/internal/runner"
	"github.com/RakhaYandra/ookami/internal/scoring"
	"github.com/spf13/cobra"
)

func NewCheckCmd() *cobra.Command {
	var jsonOut, quiet bool
	cmd := &cobra.Command{
		Use:  "check [system|development|storage|network|gpu|services]",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cat := model.Category(args[0])
			switch cat {
			case model.CategorySystem,
				model.CategoryDevelopment,
				model.CategoryStorage,
				model.CategoryNetwork,
				model.CategoryGPU,
				model.CategoryServices:
			default:
				return fmt.Errorf("invalid category %q: must be one of system|development|storage|network|gpu|services", args[0])
			}
			cfg := config.Default()
			if err := cfg.Validate(); err != nil {
				return err
			}
			r := runner.NewOSRunner(5 * time.Second)
			checks := doctor.FilterByCategory(doctor.DefaultChecks(cfg, r), cat)
			ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
			defer cancel()
			rs := doctor.RunAll(ctx, checks)
			score, status := scoring.Score(rs)
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
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output JSON")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "suppress non-essential output")
	return cmd
}
