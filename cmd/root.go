/*
Copyright © 2025 Jonas Schneider jonas.max.schneider@gmail.com
*/
package cmd

import (
	"errors"
	"os"
	"strings"

	"github.com/htwr-aachen/backend/internal/validation"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	configFile string
	logLevel   string
	conf       *viper.Viper
)

const LevelDocs = "(panic, fatal, error, warn, debug, trace)"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "htwr-aachen.de-backend",
	Short: "HTWR-Backend including exam and panikzettel streaming",
	Long:  `HTWR-Backend can query the google cloud buckets for exams and panikzettel and can send them via HTTP`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info().Msg("No subcommand given")
		return cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func parseLogLevel(levelStr string) zerolog.Level {
	level, err := zerolog.ParseLevel(levelStr)
	if err != nil {
		log.Error().Err(err).Str("log-level", levelStr).Msgf("log-level invalid. Please use %s. Defaulting to trace", LevelDocs)
		return zerolog.TraceLevel
	}

	return level
}

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.htwr-aachen.de-backend.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	initConfig()
	cobra.OnInitialize(initValidation)

	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default searches for htwr-backend.yaml in current dir, ./config, $XDG_CONFIG_HOME/htwr-backend, /etc/htwr-backend)")

	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "logging level (panic, fatal, error, warn, info, debug, trace)")

}

func initConfig() {
	conf = viper.New()
	_ = conf.BindPFlag("log_level", rootCmd.PersistentFlags().Lookup("log-level"))

	if configFile != "" {
		conf.SetConfigFile(configFile)
	} else {
		conf.SetConfigName("htwr-backend")
		conf.AddConfigPath(".")
		conf.AddConfigPath("./config")
		conf.AddConfigPath("/etc/htwr-backend")
	}

	conf.SetDefault("log_level", "Trace")

	conf.SetEnvPrefix("HTWR_BACKEND")
	conf.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	conf.AutomaticEnv()

	if err := conf.ReadInConfig(); err != nil {
		log.Info().Str("used_config_file", conf.ConfigFileUsed()).Send()
		var configFileNotFoundError viper.ConfigFileNotFoundError
		var configParseError viper.ConfigParseError

		if errors.As(err, &configFileNotFoundError) {
			log.Info().Msg("no config file found")
		} else if errors.As(err, &configParseError) {
			log.Err(err).Msg("could not parse config file... Aborting")
			cobra.CheckErr(err)
		} else {
			log.Err(err).Msg("could not load config file... Aborting")
			cobra.CheckErr(err)
		}
	}

	if conf.ConfigFileUsed() != "" {
		log.Info().Str("used_config_file", conf.ConfigFileUsed()).Send()
	}

	zerolog.SetGlobalLevel(parseLogLevel(conf.GetString("log_level")))
}

func initValidation() {
	validation.Init()
}
