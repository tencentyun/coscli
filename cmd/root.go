package cmd

import (
	"coscli/util"
	"fmt"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var config util.Config
var cmdCnt int //控制某些函数在一个命令中被调用的次数

var rootCmd = &cobra.Command{
	Use:   "coscli",
	Short: "Welcome to use coscli",
	Long:  "Welcome to use coscli!",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
	Version: "v0.10.1-beta",
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config-path", "c", "", "config file path(default is $HOME/.cos.yaml)")
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
			fmt.Println("Welcome to coscli!\nWhen you use coscli for the first time, you need to input some necessary information to generate the default configuration file of coscli.")
			initConfigFile(false)
			cmdCnt++
		}

		viper.AddConfigPath(home)
		viper.SetConfigName(".cos")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		if err := viper.UnmarshalKey("cos", &config); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
