package cmd

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	generateCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "use json as output file type")
	generateCmd.PersistentFlags().BoolVar(&htmlOut, "html", false, "use html as output file type")
	rootCmd.AddCommand(generateCmd)
}

var (
	jsonOut bool
	htmlOut bool
)

var generateCmd = &cobra.Command{
	Use:   "generate output_file_name",
	Short: "Generate findings data from database",
	Long: `Generate findings data from database. A file type can be choose either by specifying
with the flag or output file name extension (.json or .html).`,
	Args: cobra.MinimumNArgs(1),
	Run:  generate,
}

func generate(cmd *cobra.Command, args []string) {
	if jsonOut && htmlOut {
		log.Error().Msg("--json and --html flags can't be used together, choose one of them")
		os.Exit(1)
	}

	outputFileName := args[0]

	log.Debug().Str("filename", outputFileName).Msg("output")
}
