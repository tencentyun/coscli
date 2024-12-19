package cmd

import (
	"fmt"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInit(t *testing.T) {
	fmt.Println("TestConfigInit")
	//Convey("success", t, func() {
	//	cmd := rootCmd
	//	cmd.SilenceErrors = true
	//	cmd.SilenceUsage = true
	//	args := []string{"config", "init"}
	//	cmd.SetArgs(args)
	//	e := cmd.Execute()
	//	fmt.Printf(" : %v", e)
	//	So(e, ShouldBeNil)
	//})
	Convey("fail", t, func() {
		patches := ApplyFunc(initConfigFile, func(cfgFlag bool) error {
			return fmt.Errorf("test initConfigFile error")
		})
		defer patches.Reset()
		clearCmd()
		cmd := rootCmd
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		args := []string{"config", "init"}
		cmd.SetArgs(args)
		e := cmd.Execute()
		fmt.Printf(" : %v", e)
		So(e, ShouldBeError)
	})
}
