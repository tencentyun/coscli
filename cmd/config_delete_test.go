package cmd

import (
	"fmt"
	"os/exec"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConfigDeleteCmd(t *testing.T) {
	addConfig(testBucket, testEndpoint)
	Convey("Test coscil config delete", t, func() {
		Convey("success", func() {
			cmd := exec.Command("../coscli", "config", "delete", "-a",
				fmt.Sprintf("%s-%s", testBucket, appID))
			output, e := cmd.Output()
			fmt.Println(string(output))
			So(e, ShouldBeNil)
		})
		Convey("fail", func() {
			cmd := exec.Command("../coscli", "config", "delete", "-a", testAlias)
			output, e := cmd.Output()
			fmt.Println(string(output))
			So(e, ShouldBeError)
		})
	})
}
