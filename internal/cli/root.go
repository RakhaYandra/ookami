package cli

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

// Version is overridden at build time via ldflags (see Makefile).
var Version = "dev"

var noColor, verbose bool

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ookami",
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			if noColor || os.Getenv("NO_COLOR") != "" {
				lipgloss.DefaultRenderer().SetColorProfile(termenv.Ascii)
			}
		},
	}
	cmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable color output")
	cmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "verbose output")
	cmd.AddCommand(NewDoctorCmd(), NewCheckCmd(), NewVersionCmd())
	return cmd
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		_, _ = os.Stderr.WriteString("Error: " + err.Error() + "\n")
		os.Exit(4)
	}
}
