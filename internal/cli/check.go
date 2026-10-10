package cli

import (
	"fmt"

	"github.com/RakhaYandra/ookami/internal/model"
	"github.com/spf13/cobra"
)

func NewCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:  "check [system|development|storage|network|gpu|services]",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch model.Category(args[0]) {
			case model.CategorySystem,
				model.CategoryDevelopment,
				model.CategoryStorage,
				model.CategoryNetwork,
				model.CategoryGPU,
				model.CategoryServices:
				fmt.Fprintf(cmd.OutOrStdout(), "check %s not yet implemented (Phase 1)\n", args[0])
				return nil
			default:
				return fmt.Errorf("invalid category %q: must be one of system|development|storage|network|gpu|services", args[0])
			}
		},
	}
}
