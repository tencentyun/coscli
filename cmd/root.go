package cmd

import (
	_ "coscli/logger"
	"coscli/util"
	"fmt"
	"log"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var config util.Config
var param util.Param
var cmdCnt int //控制某些函数在一个命令中被调用的次数

var rootCmd = &cobra.Command{
	Use:   "coscli",
	Short: "Welcome to use coscli",
	Long:  "Welcome to use coscli!",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
	Version: "v0.11.0-beta",
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config-path", "c", "", "config file path(default is $HOME/.cos.yaml)")
	rootCmd.PersistentFlags().StringVarP(&param.SecretID, "secret-id", "i", "", "config secretId")
	rootCmd.PersistentFlags().StringVarP(&param.SecretKey, "secret-key", "k", "", "config secretKey")
	rootCmd.PersistentFlags().StringVarP(&param.SessionToken, "session-token", "t", "", "config sessionToken")
	rootCmd.PersistentFlags().StringVarP(&param.Endpoint, "endpoint", "e", "", "config endpoint")

}

func initConfig() {
	home, err := homedir.Dir()
	cobra.CheckErr(err)

	viper.SetConfigType("yaml")
	if cfgFile != "" {
		if cfgFile[0] == '~' {
			cfgFile = home + cfgFile[1:]
		}
		viper.SetConfigFile(cfgFile)
	} else {
		_, err = os.Stat(home + "/.cos.yaml")
		if os.IsNotExist(err) {
			log.Println("Welcome to coscli!\nWhen you use coscli for the first time, you need to input some necessary information to generate the default configuration file of coscli.")
			initConfigFile(false)
			cmdCnt++
		}

		viper.AddConfigPath(home)
		viper.SetConfigName(".cos")
	}

	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err == nil {
		if err := viper.UnmarshalKey("cos", &config); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if config.Base.Protocol == "" {
			config.Base.Protocol = "https"
		}
	} else {
		fmt.Println(err)
		os.Exit(1)
	}
}
