/*
Copyright © 2025 Jonas Schneider jonas.max.schneider@gmail.com
*/
package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/htwr-aachen/backend/internal/validation"
	"github.com/htwr-aachen/backend/pkg/defaults"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	configFile  string
	logLevel    string
	development bool
	conf        *koanf.Koanf
)

const logLevelDocs = "(panic, fatal, error, warn, debug, trace)"
const configFilename = "htwr-backend.yaml"
const configEnvPrefix = "HTWR_BACKEND_"

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
		log.Error().Err(err).Str("log-level", levelStr).Msgf("log-level invalid. Please use %s. Defaulting to trace", logLevelDocs)
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
	cobra.OnInitialize(initValidation)
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default searches for htwr-backend.yaml in current dir, ./config, $XDG_CONFIG_HOME/htwr-backend, /etc/htwr-backend)")

	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "logging level (panic, fatal, error, warn, info, debug, trace)")

	rootCmd.PersistentFlags().BoolVar(&development, "insecure-dev", false, "Activate insecure development mode")

}

func initConfig() {

	conf = koanf.New(".")

	if err := conf.Load(structs.Provider(defaults.GetDefaultConfig(), "koanf"), nil); err != nil {
		log.Warn().Err(err).Msg("Failed to load default config")
	}

	loadConfigFile()

	if err := conf.Load(env.Provider(".", env.Opt{
		Prefix:        configEnvPrefix,
		TransformFunc: transformEnvKey,
	}), nil); err != nil {
		log.Warn().Err(err).Msg("Failed to load environment variables")
	}

	flagSet := rootCmd.PersistentFlags()
	if err := conf.Load(posflag.ProviderWithFlag(flagSet, ".", conf, func(f *pflag.Flag) (string, any) {
		k := f.Name
		switch k {
		case "config":
			return "", ""
		case "log-level":
			return "global.log_level", posflag.FlagVal(flagSet, f)
		case "insecure-dev":
			return "global.insecure_dev", posflag.FlagVal(flagSet, f)
		default:
			return k, posflag.FlagVal(flagSet, f)
		}
	}), nil); err != nil {
		log.Warn().Err(err).Msg("Failed to load command-line flags")
	}
	zerolog.SetGlobalLevel(parseLogLevel(conf.String("global.log_level")))
}

func initValidation() {
	validation.Init()
}

func loadConfigFile() {
	configPaths := getConfigPaths()

	for _, path := range configPaths {
		if path == "" {
			continue
		}

		err := conf.Load(file.Provider(path), yaml.Parser())
		if err == nil {
			log.Info().Str("config_file", path).Msg("Loaded config file")
			return
		}

		// If user explicitly specified a config file and it fails, exit
		if configFile != "" && path == configFile {
			log.Fatal().Str("config_file", path).Err(err).Msg("Config file specified but not loadable")
		}

		log.Debug().Str("path", path).Err(err).Msg("Config file not found or error loading")
	}

	log.Info().Msg("No config file loaded, using defaults and environment variables")
}

func getConfigPaths() []string {
	paths := make([]string, 0, 5)

	// User-specified config file try if specified and only try this.
	if configFile != "" {
		paths = append(paths, configFile)
		return paths
	}

	// Current directory
	if currentDir, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(currentDir, configFilename))
	}

	// User config directory
	if userConfigDir, err := os.UserConfigDir(); err == nil {
		paths = append(paths, filepath.Join(userConfigDir, "htwr-backend", configFilename))
	}

	// System config directory
	paths = append(paths, filepath.Join("/etc/htwr-backend", configFilename))

	return paths
}

func transformEnvKey(k, v string) (string, any) {
	// Remove prefix and convert to lowercase with dots
	k = strings.ReplaceAll(
		strings.ToLower(strings.TrimPrefix(k, configEnvPrefix)),
		"_",
		".",
	)

	// Split space-separated values into arrays
	if strings.Contains(v, " ") {
		return k, strings.Split(v, " ")
	}

	return k, v
}
