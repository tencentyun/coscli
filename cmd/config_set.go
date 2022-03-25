package cmd

import (
	"os"

	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Used to modify configuration items in the [base] group of the configuration file",
	Long: `Used to modify configuration items in the [base] group of the configuration file

Format:
  ./coscli config set [flags]

Example:
  ./coscli config set -t example-token`,
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		setConfigItem(cmd)
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)

	configSetCmd.Flags().StringP("secret_id", "", "", "Set secret id")
	configSetCmd.Flags().StringP("secret_key", "", "", "Set secret key")
	configSetCmd.Flags().StringP("session_token", "", "", "Set session token")
}

func setConfigItem(cmd *cobra.Command) {
	flag := false
	secretID, _ := cmd.Flags().GetString("secret_id")
	secretKey, _ := cmd.Flags().GetString("secret_key")
	sessionToken, _ := cmd.Flags().GetString("session_token")

	if secretID != "" {
		flag = true
		if secretID == "@" {
			config.Base.SecretID = ""
		} else {
			config.Base.SecretID = secretID
		}
	}
	if secretKey != "" {
		flag = true
		if secretKey == "@" {
			config.Base.SecretKey = ""
		} else {
			config.Base.SecretKey = secretKey
		}
	}
	if sessionToken != "" {
		flag = true
		if sessionToken == "@" {
			config.Base.SessionToken = ""
		} else {
			config.Base.SessionToken = sessionToken
		}
	}
	if !flag {
		logger.Fatalln("Enter at least one configuration item to be modified!")
		logger.Infoln(cmd.UsageString())
		os.Exit(1)
	}

	viper.Set("cos.base", config.Base)
	if err := viper.WriteConfigAs(viper.ConfigFileUsed()); err != nil {
		logger.Fatalln(err)
		os.Exit(1)
	}
	logger.Infoln("Modify successfully!")
}
