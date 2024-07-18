package cmd

import (
	"coscli/util"
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func getSet(oldconfig *util.Config) {
	oldconfig.Base = config.Base
	oldconfig.Base.SecretID, _ = util.DecryptSecret(config.Base.SecretID)
	oldconfig.Base.SecretKey, _ = util.DecryptSecret(config.Base.SecretKey)
	oldconfig.Base.SessionToken, _ = util.DecryptSecret(config.Base.SessionToken)
}

func TestConfigSetCmd(t *testing.T) {
	fmt.Println("TestConfigSetCmd")
	oldconfig := &util.Config{}
	getSet(oldconfig)
	cmd := rootCmd
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	Convey("Test coscil config set", t, func() {
		Convey("fail", func() {
			Convey("Invalid argument", func() {
				args := []string{"config", "set", "--secret_id"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeError)
			})
			Convey("@", func() {
				args := []string{"config", "set", "--secret_id", "@",
					"--secret_key", "@", "--session_token", "@", "--mode", "@",
					"--cvm_role_name", "@", "--close_auto_switch_host", "@"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeError)
			})
			Convey("no arguments", func() {
				args := []string{"config", "set"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeError)
			})
		})
		Convey("success", func() {
			Convey("give arguments", func() {
				args := []string{"config", "set", "--secret_id", oldconfig.Base.SecretID,
					"--secret_key", oldconfig.Base.SecretKey, "--session_token", oldconfig.Base.SessionToken, "--mode", oldconfig.Base.Mode,
					"--cvm_role_name", oldconfig.Base.CvmRoleName, "--close_auto_switch_host", oldconfig.Base.CloseAutoSwitchHost}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})

	})
	time.Sleep(1 * time.Second)
}
