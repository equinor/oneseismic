package cmd

import (
	"github.com/equinor/seismic-cloud-api/api/config"
	l "github.com/equinor/seismic-cloud-api/api/logger"
	"github.com/spf13/cobra"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "api",
	Short: "A server wrapper for seismic cloud",
	Long:  `A server wrapper for seismic cloud`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute(app, v string) {
	rootCmd.Use = app
	rootCmd.Version = v

	if err := rootCmd.Execute(); err != nil {
		l.LogE("Executing command", err)
		panic(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .sc-api.yaml)")
}

func initConfig() {
	if err := config.Init(cfgFile); err != nil {
		l.LogE("Init config", err)
		panic(err)
	}
}
