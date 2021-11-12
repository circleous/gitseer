package cmd

import (
	"os"

	"github.com/circleous/gitseer/internal/analysis"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(scanCmd)
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Start scan for secrets",
	Long:  `Start scan for secrets in git repositories`,
	Run:   scan,
}

func scan(cmd *cobra.Command, args []string) {
	conf, err := analysis.ParseConfig(confPath)
	if err != nil {
		log.Error().Err(err).Msg("failed to parse config file")
		os.Exit(1)
	}

	a, err := analysis.New(conf)
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize")
		os.Exit(1)
	}

	a.Runner()
}
