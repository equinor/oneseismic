package cmd

import (
	"os"

	l "github.com/equinor/seismic-cloud/api/logger"

	"github.com/equinor/seismic-cloud/api/config"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "api",
	Short: "A server wrapper for seismic cloud",
	Long:  `A server wrapper for seismic cloud`,
}

// Eecute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		l.LogE("root.Execute", "", err)
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
			l.LogE("root.initConfig", "Open working dir", err)
			os.Exit(1)
		}
		viper.AddConfigPath(wd)
		viper.SetConfigName(".sc-api")
	}
	config.SetDefaults()
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err == nil {
		l.LogI("root.initConfig", "Using config file")
	}
	if err := config.Load(); err == nil {
		l.LogI("root.initConfig", "Config loaded and validated "+viper.ConfigFileUsed())
	}
}
