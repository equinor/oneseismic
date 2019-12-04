package cmd

import (
	"os"

	l "github.com/equinor/seismic-cloud-api/api/logger"

	"github.com/equinor/seismic-cloud-api/api/config"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .sc-api.yaml)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		wd, err := os.Getwd()
		if err != nil {
			l.LogE("Open working dir", err)
			os.Exit(1)
		}
		viper.AddConfigPath(wd)
		viper.SetConfigName(".sc-api")
	}
	config.SetDefaults()
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err == nil {
		l.LogI("Reading config file")
	}
	if err := config.Load(); err == nil {
		l.LogI("Config loaded and validated " + viper.ConfigFileUsed())
	}
}
