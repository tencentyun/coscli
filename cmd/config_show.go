package cmd

import (
	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Prints information from a specified configuration file",
	Long: `Prints information from a specified configuration file

Format:
  ./coscli config show [-c <config-file-path>]

Example:
  ./coscli config show`,
	Run: func(cmd *cobra.Command, args []string) {
		showConfig()
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
}

func showConfig() {
	logger.Infoln("Configuration file path:")
	logger.Infof("  %s\n", viper.ConfigFileUsed())
	logger.Infoln("====================")
	logger.Infoln("Basic Configuration Information:")
	logger.Infof("  Secret ID:     %s\n", config.Base.SecretID)
	logger.Infof("  Secret Key:    %s\n", config.Base.SecretKey)
	logger.Infof("  Session Token: %s\n", config.Base.SessionToken)
	logger.Infoln("====================")
	logger.Infoln("Bucket Configuration Information:")

	for i, b := range config.Buckets {
		logger.Infof("- Bucket %d :\n", i+1)
		logger.Infof("  Name:  \t%s\n", b.Name)
		logger.Infof("  Endpoint:\t%s\n", b.Endpoint)
		logger.Infof("  Alias: \t%s\n", b.Alias)
		logger.Infof(" Ofs: \t%v\n", b.Ofs)
	}
}
