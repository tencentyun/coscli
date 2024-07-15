package cmd

import (
	"fmt"
	"os/exec"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConfigShowCmd(t *testing.T) {
	Convey("Test coscil config show", t, func() {
		Convey("success", func() {
			cmd := exec.Command("../coscli", "config", "show")
			output, e := cmd.Output()
			fmt.Println(string(output))
			So(e, ShouldBeNil)
		})
	})
}
