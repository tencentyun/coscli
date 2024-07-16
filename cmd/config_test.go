package cmd

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConfigCmd(t *testing.T) {
	fmt.Println("TestConfigCmd")
	Convey("Test coscli config", t, func() {
		cmd := rootCmd
		args := []string{"config"}
		cmd.SetArgs(args)
		e := cmd.Execute()
		So(e, ShouldBeNil)
	})
}
