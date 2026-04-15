package cmd

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConfigShowCmd(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("Test coscli config show", t, func() {
		Reset(func() {
			clearCmd()
		})

		Convey("show config", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "show", "-c", testConfigPath})
			e := cmd.Execute()
			fmt.Printf(" : %v", e)
			So(e, ShouldBeNil)
		})
	})
}
