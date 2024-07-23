package cmd

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRbCmd(t *testing.T) {
	fmt.Println("TestRbCmd")
	testBucket = randStr(8)
	// 仅创建桶，不添加配置
	setUp(testBucket, "nil", testEndpoint, false)
	clearCmd()
	cmd := rootCmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	Convey("Test coscli rb", t, func() {
		Convey("success", func() {
			Convey("force", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"rb",
					fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint, "-f"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})
		Convey("fail", func() {
			Convey("Not enough arguments", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"rb"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("Invalid bukcetIDName", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"rb", "cos:/"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("Not exist", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"rb",
					fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
		})
	})
}
