package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewDoctorCmd() *cobra.Command {
	var fix, jsonOut, quiet bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "full health check",
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "doctor not yet implemented (Phase 1)")
			return nil
		},
	}
	cmd.Flags().BoolVar(&fix, "fix", false, "attempt safe fixes")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output JSON")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "suppress non-essential output")
	return cmd
}
