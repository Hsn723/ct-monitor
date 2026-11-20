package cmd

import (
	"log/slog"
	"os"

	"github.com/Hsn723/ct-monitor/config"
	"github.com/Hsn723/ct-monitor/monitor"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:           "ct-monitor",
		Short:         "ct-monitor queries the certspotter API for new certificate issuances",
		RunE:          runRoot,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	configFile string

	version string
	commit  string
	date    string
	builtBy string

	fs = afero.NewOsFs()

	ctMonitor monitor.IMonitor
)

func init() {
	rootCmd.Flags().StringVarP(&configFile, "config", "c", config.DefaultConfigFile, "path to configuration file")
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))
	slog.SetDefault(logger)
}

func runRoot(_ *cobra.Command, _ []string) error {
	slog.Info("ct-monitor", "version", version, "commit", commit, "date", date, "built_by", builtBy)
	conf, err := config.Load(fs, configFile)
	if err != nil {
		return err
	}
	slog.Info("loaded configuration", "config", configFile)
	if ctMonitor == nil {
		ctMonitor = monitor.NewMonitor(conf, fs)
	}
	return ctMonitor.Run()
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		slog.Error("ct-monitor encountered an error", "error", err.Error())
		os.Exit(1)
	}
}
