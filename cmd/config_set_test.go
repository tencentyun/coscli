package cmd

import (
	"coscli/util"
	"fmt"
	"os/exec"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func getSet(oldconfig *util.Config) {
	oldconfig.Base = config.Base
	oldconfig.Base.SecretID, _ = util.DecryptSecret(config.Base.SecretID)
	oldconfig.Base.SecretKey, _ = util.DecryptSecret(config.Base.SecretKey)
	oldconfig.Base.SessionToken, _ = util.DecryptSecret(config.Base.SessionToken)
}

func TestConfigSetCmd(t *testing.T) {
	oldconfig := &util.Config{}
	getSet(oldconfig)
	Convey("Test coscil config set", t, func() {
		Convey("fail", func() {
			Convey("Invalid argument", func() {
				cmd := exec.Command("../coscli", "config", "set", "--secret_id")
				output, e := cmd.Output()
				fmt.Println(string(output))
				So(e, ShouldBeError)
			})
			Convey("@", func() {
				cmd := exec.Command("../coscli", "config", "set", "--secret_id", "@",
					"--secret_key", "@", "--session_token", "@", "--mode", "@",
					"--cvm_role_name", "@", "--close_auto_switch_host", "@")
				output, e := cmd.Output()
				fmt.Println(string(output))
				So(e, ShouldBeError)
			})
		})
		Convey("success", func() {
			Convey("no arguments", func() {
				cmd := exec.Command("../coscli", "config", "set")
				output, e := cmd.Output()
				fmt.Println(string(output))
				So(e, ShouldBeNil)
			})
			Convey("give arguments", func() {
				cmd := exec.Command("../coscli", "config", "set", "--secret_id", oldconfig.Base.SecretID,
					"--secret_key", oldconfig.Base.SecretKey, "--session_token", oldconfig.Base.SessionToken, "--mode", oldconfig.Base.Mode,
					"--cvm_role_name", oldconfig.Base.CvmRoleName, "--close_auto_switch_host", oldconfig.Base.CloseAutoSwitchHost)
				output, e := cmd.Output()
				fmt.Println(string(output))
				So(e, ShouldBeNil)
			})
		})

	})
}
