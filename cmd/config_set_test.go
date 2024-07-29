package cmd

import (
	"coscli/util"
	"fmt"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/viper"
)

func TestConfigSetCmd(t *testing.T) {
	fmt.Println("TestConfigSetCmd")
	getConfig()
	var oldconfig util.Config = config
	secretKey, err := util.DecryptSecret(config.Base.SecretKey)
	if err == nil {
		oldconfig.Base.SecretKey = secretKey
	}
	secretId, err := util.DecryptSecret(config.Base.SecretID)
	if err == nil {
		oldconfig.Base.SecretID = secretId
	}
	sessionToken, err := util.DecryptSecret(config.Base.SessionToken)
	if err == nil {
		oldconfig.Base.SessionToken = sessionToken
	}
	clearCmd()
	cmd := rootCmd
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	Convey("Test coscil config set", t, func() {
		Convey("fail", func() {
			// Convey("Invalid argument", func() {
			// 	args := []string{"config", "set", "--secret_id"}
			// 	cmd.SetArgs(args)
			// 	e := cmd.Execute()
			// 	fmt.Printf(" : %v", e)
			// 	So(e, ShouldBeError)
			// })
			Convey("no arguments", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"config", "set"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("no mode", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"config", "set", "--mode", "@"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("@", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(viper.WriteConfigAs, func(string) error {
					return fmt.Errorf("test WriteConfigAs fail")
				})
				defer patches.Reset()
				args := []string{"config", "set", "--secret_id", "@",
					"--secret_key", "@", "--session_token", "@", "--mode", "",
					"--cvm_role_name", "@", "--close_auto_switch_host", "@"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("token,mode,cvm", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(viper.WriteConfigAs, func(string) error {
					return fmt.Errorf("test WriteConfigAs fail")
				})
				defer patches.Reset()
				args := []string{"config", "set", "--session_token", "666", "--mode", "CvmRole",
					"--cvm_role_name", "name"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("cfgFile", func() {
				patches := ApplyFunc(viper.WriteConfigAs, func(string) error {
					return fmt.Errorf("test write configas error")
				})
				defer patches.Reset()
				clearCmd()
				cmd := rootCmd
				args := []string{"config", "set", "--secret_id", "@", "-c", "./test.yaml"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
		})
		Convey("success", func() {
			Convey("give arguments", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"config", "set", "--secret_id", oldconfig.Base.SecretID,
					"--secret_key", oldconfig.Base.SecretKey, "--session_token", "@", "--mode", oldconfig.Base.Mode,
					"--cvm_role_name", "@", "--close_auto_switch_host", oldconfig.Base.CloseAutoSwitchHost}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})

	})
}
