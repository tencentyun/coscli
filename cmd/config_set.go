package cmd

import (
	"coscli/util"
	"fmt"
	"github.com/mitchellh/go-homedir"
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
	configSetCmd.Flags().StringP("session_token", "t", "", "Set session token")
	configSetCmd.Flags().StringP("mode", "", "", "Set mode")
	configSetCmd.Flags().StringP("cvm_role_name", "", "", "Set cvm role name")
	configSetCmd.Flags().StringP("close_auto_switch_host", "", "", "Close Auto Switch Host")
}

func setConfigItem(cmd *cobra.Command) {
	flag := false
	secretID, _ := cmd.Flags().GetString("secret_id")
	secretKey, _ := cmd.Flags().GetString("secret_key")
	sessionToken, _ := cmd.Flags().GetString("session_token")
	mode, _ := cmd.Flags().GetString("mode")
	cvmRoleName, _ := cmd.Flags().GetString("cvm_role_name")
	closeAutoSwitchHost, _ := cmd.Flags().GetString("close_auto_switch_host")
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
	if mode != "" {
		flag = true
		if mode != "SecretKey" && mode != "CvmRole" {
			logger.Fatalln("Please Enter Mode As SecretKey Or CvmRole!")
			logger.Infoln(cmd.UsageString())
			os.Exit(1)
		} else {
			config.Base.Mode = mode
		}
	}
	if cvmRoleName != "" {
		flag = true
		if cvmRoleName == "@" {
			config.Base.CvmRoleName = ""
		} else {
			config.Base.CvmRoleName = cvmRoleName
		}
	}

	if closeAutoSwitchHost != "" {
		flag = true
		if closeAutoSwitchHost == "@" {
			config.Base.CloseAutoSwitchHost = ""
		} else {
			config.Base.CloseAutoSwitchHost = closeAutoSwitchHost
		}
	}

	if !flag {
		logger.Fatalln("Enter at least one configuration item to be modified!")
		logger.Infoln(cmd.UsageString())
		os.Exit(1)
	}
	// 默认加密存储
	config.Base.SecretKey, _ = util.EncryptSecret(config.Base.SecretKey)
	config.Base.SecretID, _ = util.EncryptSecret(config.Base.SecretID)
	config.Base.SessionToken, _ = util.EncryptSecret(config.Base.SessionToken)

	// 判断config文件是否存在。不存在则创建
	home, err := homedir.Dir()
	configFile := ""
	if cfgFile != "" {
		if cfgFile[0] == '~' {
			configFile = home + cfgFile[1:]
		} else {
			configFile = cfgFile
		}
	} else {
		configFile = home + "/.cos.yaml"
	}
	_, err = os.Stat(configFile)
	if os.IsNotExist(err) || cfgFile != "" {
		viper.Set("cos.base", config.Base)
		if err := viper.WriteConfigAs(configFile); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	} else {
		viper.Set("cos.base", config.Base)
		if err := viper.WriteConfigAs(viper.ConfigFileUsed()); err != nil {
			logger.Fatalln(err)
			os.Exit(1)
		}
	}

	logger.Infoln("Modify successfully!")
}
