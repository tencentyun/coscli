package cmd

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRmCmd(t *testing.T) {
	cmd := rootCmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	Convey("Test coscli rm", t, func() {
		Convey("Not enough arguments", func() {
			clearCmd()
			cmd := rootCmd
			args := []string{"rm"}
			cmd.SetArgs(args)
			e := cmd.Execute()
			fmt.Printf(" : %v", e)
			So(e, ShouldBeError)
		})
		Convey("Invalid arguments", func() {
			clearCmd()
			cmd := rootCmd
			args := []string{"rm", "invaild"}
			cmd.SetArgs(args)
			e := cmd.Execute()
			fmt.Printf(" : %v", e)
			So(e, ShouldBeError)
		})
	})
}
