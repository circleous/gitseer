package cmd

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gitseer",
	Short: "gitseer is a tool to scan for secrets in git repositories",
	Long:  `A flexible secrets scanner for git repositories. Currently supports gitlab and github.`,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

var (
	confPath string
	dbPath   string
	silent   bool
	verbose  bool
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&confPath, "config", "c", "gitseer.toml", "config file")
	rootCmd.PersistentFlags().BoolVarP(&silent, "silent", "s", false, "silent, only error or panic output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "more verbose for debug output")
	cobra.OnInitialize(func() {
		// init logger
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

		if silent && verbose {
			log.Error().Msg("choose only one of silent or verbose output")
			os.Exit(1)
		}

		zerolog.SetGlobalLevel(zerolog.InfoLevel)

		if silent {
			zerolog.SetGlobalLevel(zerolog.ErrorLevel)
		}

		if verbose {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		}

		if _, err := os.Stat(confPath); os.IsNotExist(err) {
			log.Error().Err(err).Msg("config file not exists!")
			os.Exit(1)
		}
	})
}

// Execute root cobra executor
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Error().Err(err).Msg("")
		os.Exit(1)
	}
}
