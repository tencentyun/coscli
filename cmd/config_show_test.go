package cmd

import (
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConfigShowCmd(t *testing.T) {
	fmt.Println("TestConfigShowCmd")
	Convey("Test coscil config show", t, func() {
		Convey("success", func() {
			Convey("give arguments", func() {
				cmd := rootCmd
				args := []string{"config", "show"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})
	})
	time.Sleep(1 * time.Second)
}
