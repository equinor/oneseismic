package cmd

import (
	"fmt"

	"github.com/equinor/seismic-cloud/api/config"
	l "github.com/equinor/seismic-cloud/api/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(defaultsCmd)
}

var defaultsCmd = &cobra.Command{
	Use:   "defaults",
	Short: "Writes default config values to config file",
	Long:  `Writes default config values to config file, specified with --config`,
	Run:   createDefaults,
}

func createDefaults(cmd *cobra.Command, args []string) {
	config.SetDefaults()
	viper.AutomaticEnv()
	err := viper.WriteConfig()
	if err == nil {
		s := fmt.Sprintf("Writing default config file to %s", viper.ConfigFileUsed())
		l.LogI("defaults.createDefaults", s)
	}
}
