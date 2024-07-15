package cmd

import (
	"fmt"
	"os/exec"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConfigCmd(t *testing.T) {
	Convey("Test coscli config", t, func() {
		cmd := exec.Command("../coscli", "config")
		output, e := cmd.Output()
		fmt.Println(string(output))
		So(e, ShouldBeNil)
	})
}
