package cmd

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/circleous/gitseer/internal/analysis"
	"github.com/circleous/gitseer/pkg/signature"
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

func scan(_ *cobra.Command, _ []string) {
	conf, err := analysis.ParseConfig(confPath)
	if err != nil {
		log.Error().Err(err).Msg("failed to parse config file")
		os.Exit(1)
	}

	sig, err := signature.LoadSignature(conf.SignaturePath)
	if err != nil {
		log.Error().Err(err).Msg("failed to signature")
		os.Exit(1)
	}

	a, err := analysis.New(conf, sig.Signatures)
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize")
		os.Exit(1)
	}

	a.Runner()
}
