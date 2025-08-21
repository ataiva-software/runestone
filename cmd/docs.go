package cmd

import (
	"fmt"

	"github.com/ataiva-software/runestone/internal/docs"
	"github.com/spf13/cobra"
)

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Generate documentation",
	Long: `Generate comprehensive documentation including:
- Getting Started guide
- API Reference
- Configuration Reference  
- Examples`,
	RunE: runDocs,
}

func init() {
	docsCmd.Flags().StringP("output", "o", "docs", "Output directory for documentation")
}

func runDocs(cmd *cobra.Command, args []string) error {
	outputDir, _ := cmd.Flags().GetString("output")

	fmt.Println("Generating documentation...")

	generator := docs.NewGenerator(outputDir)
	if err := generator.Generate(); err != nil {
		return fmt.Errorf("failed to generate documentation: %w", err)
	}

	fmt.Printf("Documentation generated in %s\n", outputDir)
	return nil
}
