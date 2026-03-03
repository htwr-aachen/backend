/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/htwr-aachen/backend/internal/configurator"
	"github.com/htwr-aachen/backend/internal/database"
	"github.com/htwr-aachen/backend/internal/instrumentation"
	"github.com/htwr-aachen/backend/internal/metrics"
	"github.com/htwr-aachen/backend/internal/server"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run htwr backend server",
	Long: `Start the htwr-aachen backend server with configured services.

Available Services:
- QA API: Question and Answer service (/api/qa)
- Panikzettel API: Emergency sheet service (/api/panikzettel)
- Admin Interface: Administration web interface and API

Configuration Priority (highest to lowest):
1. Command line flags (--qa-database-url, --panikzettel-storage-path, etc.)
2. Environment variables (HTWR_QA_DATABASE_URL, HTWR_PANIKZETTEL_STORAGE_PATH, etc.)
3. Configuration file values
4. Default values

The main API services run on --port (default: 8080)
The admin interface runs on --admin-port (default: 8081) for security isolation.

Examples:
  # Run with defaults
  htwr-backend run

  # Custom ports
  htwr-backend run --port 9000 --admin-port 9001

  # With config file
  htwr-backend run --config /etc/htwr-backend/config.yaml

  # With environment variables
  HTWR_QA_DATABASE_URL=postgres://... htwr-backend run

  # Disable specific services
  htwr-backend run --no-admin --qa-database-url postgres://...

  # Mixed configuration
  HTWR_ADMIN_AUTH_SECRET=secret123 htwr-backend run --config config.yaml --port 9000`,
	RunE: func(cmd *cobra.Command, args []string) error {

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		ctx, err := configurator.LoadAndAttach(ctx, conf)
		if err != nil {
			return fmt.Errorf("loading & validating configuration: %w", err)
		}

		cfg, ok := configurator.FromContext(ctx)
		if !ok {
			return fmt.Errorf("no configuration in context")
		}

		// Initialize OpenTelemetry
		im, err := instrumentation.Start(ctx, &cfg.Global.OpenTelemetry, cfg.Global.OpenTelemetry.ServiceName)
		if err != nil {
			log.Warn().Err(err).Msg("failed to initialize OpenTelemetry")
		} else if cfg.Global.OpenTelemetry.Enabled {
			log.Info().Msg("OpenTelemetry initialized")
			defer func() {
				if err := im.Shutdown(ctx); err != nil {
					log.Error().Err(err).Msg("error shutting down OpenTelemetry")
				}
			}()
		}
		ctx = instrumentation.AttachToContext(ctx, im)

		ctx, err = metrics.CreateAndAttach(ctx)
		if err != nil {
			log.Error().Err(err).Msg("creating metrics recorder")
			return fmt.Errorf("could not create metrics recorder: %w", err)
		}

		ctx, err = database.CreateAndAttach(ctx)
		if err != nil {
			log.Error().Err(err).Msg("creating database connection pool")

			return fmt.Errorf("could not connect to database %w", err)
		}
		defer database.Close()

		server, err := server.New(ctx)
		if err != nil {
			log.Error().Err(err).Msg("creating server structure")
			return err
		}

		err = server.Run(ctx)
		if err != nil {
			log.Error().Err(err).Msg("running application")
			return err
		}

		return nil
	},
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// runCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// runCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	runCmd.PersistentFlags().Bool("qa-disabled", false, "Enable the qa subsystem")
	runCmd.PersistentFlags().Bool("panikzettel-disabled", false, "Enable the panikzettel subsystem")
	runCmd.PersistentFlags().String("public-host", "", "hostname/ip to bind the public public server to. Defaults to [::]")
	runCmd.PersistentFlags().String("public-port", "", "port to bind the public public service to. Defaults to 8080")

	runCmd.PersistentFlags().Bool("admin-disabled", false, "Enable the admin subsystem")
	runCmd.PersistentFlags().String("admin-host", "", "Hostname/IP to bind the admin endpoints to. Defaults to [::]")
	runCmd.PersistentFlags().String("admin-port", "", "port to bind the admin service to. Combining with others, if full hostname:port matches. Defaults to 8081")

	runCmd.PersistentFlags().Bool("metrics-disabled", false, "Enable the metrics subsystem")
	runCmd.PersistentFlags().String("metrics-host", "", "Hostname/IP to bind the metrics endpoints to. Defaults to [::]")
	runCmd.PersistentFlags().String("metrics-port", "", "Port to bind the metrics endpoints to. Combining with others, if full hostname:port matches. Defaults to 9090")

	cobra.OnInitialize(runBind)
}

func runBind() {

	fs := runCmd.Flags()
	if err := conf.Load(posflag.ProviderWithFlag(fs, ".", conf, func(f *pflag.Flag) (string, any) {
		if k, found := strings.CutSuffix(f.Name, "-disabled"); found {
			val, ok := posflag.FlagVal(fs, f).(bool)
			if !ok {
				log.Panic().Stack().Msg("flag not typed as expected")
			}

			k := strings.ReplaceAll(k, "-", ".") + ".enabled"

			return k, !val
		}

		return strings.ReplaceAll(f.Name, "-", "."), posflag.FlagVal(fs, f)

	}), nil); err != nil {
		log.Fatal().Err(err).Stack().Msg("binding flags into configuration")
	}
}
