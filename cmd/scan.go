package cmd

import (
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/circleous/gitseer/internal/analysis"
	"github.com/circleous/gitseer/pkg/signature"
)

func init() {
	scanCmd.PersistentFlags().StringVarP(&generatedFileName, "output", "o", "",
		"generate a .json or .html report file after scan")
	rootCmd.AddCommand(scanCmd)
}

var (
	generatedFileName string
)

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
	defer a.Close()

	a.Runner()

	if generatedFileName != "" {
		if strings.HasSuffix(generatedFileName, ".json") {

		} else if strings.HasSuffix(generatedFileName, ".html") {

		} else {
			log.Error().Msg("invalid file type")
			os.Exit(1)
		}
	}
}
