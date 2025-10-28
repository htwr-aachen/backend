/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/htwr-aachen/backend/internal/configurator"
	"github.com/htwr-aachen/backend/internal/database"
	"github.com/htwr-aachen/backend/internal/server"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
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
		server, err := server.New(conf)
		if err != nil {
			log.Error().Err(err).Msg("validating configuration")
			return err
		}
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

		ctx, err = configurator.LoadAndAttach(ctx, conf)
		if err != nil {
			log.Error().Err(err).Msg("loading global configuration")
			return fmt.Errorf("could not load global configuration: %w", err)
		}
		ctx, err = database.CreateAndAttach(ctx, conf)
		if err != nil {
			log.Error().Err(err).Msg("creating database connection pool")

			return fmt.Errorf("could not connect to database %w", err)
		}

		defer cancel()
		defer database.Close()
		err = server.Run(ctx, conf)
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

	runCmd.PersistentFlags().String("public-host", "", "hostname/ip to bind the public server to. Defaults to [::]")
	_ = conf.BindPFlag("qa.host", runCmd.PersistentFlags().Lookup("public-host"))
	runCmd.PersistentFlags().String("public-port", "", "port to bind the public service to. Defaults to 8080")
	_ = conf.BindPFlag("qa.port", runCmd.PersistentFlags().Lookup("public-port"))

	runCmd.PersistentFlags().String("admin-host", "", "Hostname/IP to bind the admin endpoints to. Defaults to [::]")
	_ = conf.BindPFlag("admin.host", runCmd.PersistentFlags().Lookup("admin-host"))
	runCmd.PersistentFlags().String("admin-port", "", "port to bind the admin service to. Combining with others, if full hostname:port matches. Defaults to 8081")
	_ = conf.BindPFlag("admin.port", runCmd.PersistentFlags().Lookup("admin-port"))

	runCmd.PersistentFlags().String("metrics-host", "", "Hostname/IP to bind the metrics endpoints to. Defaults to [::]")
	_ = conf.BindPFlag("metrics.host", runCmd.PersistentFlags().Lookup("metrics-host"))
	runCmd.PersistentFlags().String("metrics-port", "", "Port to bind the metrics endpoints to. Combining with others, if full hostname:port matches. Defaults to 9090")
	_ = conf.BindPFlag("metrics_port", runCmd.PersistentFlags().Lookup("metrics-port"))

	runCmd.PersistentFlags().Bool("disable-panikzettel", false, "Disable the Panikzettel subsystem")
	_ = conf.BindPFlag("panikzettel.disabled", runCmd.PersistentFlags().Lookup("disable-panikzettel"))
	runCmd.PersistentFlags().Bool("disable-qa", false, "Disable the QA subsystem")
	_ = conf.BindPFlag("qa.disabled", runCmd.PersistentFlags().Lookup("disable-qa"))
	runCmd.PersistentFlags().Bool("disable-admin", false, "Disable the Admin subsystem")
	_ = conf.BindPFlag("admin.disabled", runCmd.PersistentFlags().Lookup("disable-admin"))
}
