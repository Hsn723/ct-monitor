package cmd

import (
	"log/slog"
	"os"

	"github.com/Hsn723/ct-monitor/config"
	"github.com/Hsn723/ct-monitor/monitor"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

type app struct {
	fs         afero.Fs
	configFile string
	monitor    monitor.IMonitor
}

var (
	version string
	commit  string
	date    string
	builtBy string
)

// RootCmd returns the root command.
func (a *app) RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ct-monitor",
		Short:         "ct-monitor queries the certspotter API for new certificate issuances",
		RunE:          a.runRoot,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.Flags().StringVarP(&a.configFile, "config", "c", config.DefaultConfigFile, "path to configuration file")
	return cmd
}

func init() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))
	slog.SetDefault(logger)
}

func (a *app) runRoot(_ *cobra.Command, _ []string) error {
	slog.Info("ct-monitor", "version", version, "commit", commit, "date", date, "built_by", builtBy)
	conf, err := config.Load(a.fs, a.configFile)
	if err != nil {
		return err
	}
	slog.Info("loaded configuration", "config", a.configFile)
	if a.monitor == nil {
		a.monitor = monitor.NewMonitor(conf, a.fs)
	}
	return a.monitor.Run()
}

// Execute runs the root command.
func Execute() {
	app := &app{
		fs: afero.NewOsFs(),
	}
	if err := app.RootCmd().Execute(); err != nil {
		slog.Error("ct-monitor encountered an error", "error", err.Error())
		os.Exit(1)
	}
}
