package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "runestone",
	Short: "Runestone - Declarative, drift-aware infrastructure",
	Long: `Runestone is Ataiva's next-generation Infrastructure-as-Code platform.
It solves the common pain points of existing IaC tools — brittle state files,
drift surprises, and complex multi-cloud orchestration — by offering a stateless,
DAG-driven execution engine with real-time reconciliation and human-friendly CLI workflows.`,
}

func SetVersion(version string) {
	rootCmd.Version = version
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(bootstrapCmd)
	rootCmd.AddCommand(previewCmd)
	rootCmd.AddCommand(commitCmd)
	rootCmd.AddCommand(alignCmd)
	rootCmd.AddCommand(dismantleCmd)
	rootCmd.AddCommand(docsCmd)
}
