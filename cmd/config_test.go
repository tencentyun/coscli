package cmd

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConfigCmd(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("Test coscli config", t, func() {
		Reset(func() {
			clearCmd()
		})

		Convey("show help", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"config"})
			e := cmd.Execute()
			fmt.Printf(" : %v", e)
			So(e, ShouldBeNil)
		})
	})
}
