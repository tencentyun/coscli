package cmd

import (
	"coscli/util"
	"fmt"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tencentyun/cos-go-sdk-v5"
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
			Convey("Endpoint", func() {
				patches := ApplyFunc(removeBucket, func(string) error {
					return fmt.Errorf("test removeBucket error")
				})
				defer patches.Reset()
				clearCmd()
				cmd := rootCmd
				args := []string{"rb", fmt.Sprintf("cos://%s-%s", testBucket, appID), "--region", "guangzhou"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("RemoveObjects", func() {
				patches := ApplyFunc(util.RemoveObjects, func(args []string, fo *util.FileOperations) error {
					return fmt.Errorf("test RemoveObjects error")
				})
				defer patches.Reset()
				clearCmd()
				cmd := rootCmd
				args := []string{"rb",
					fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint, "-f"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("abortParts", func() {
				patches := ApplyFunc(abortParts, func(arg string, include string, exclude string) error {
					return fmt.Errorf("test abortParts error")
				})
				defer patches.Reset()
				clearCmd()
				cmd := rootCmd
				args := []string{"rb",
					fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint, "-f"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})

		})
		Convey("success and again", func() {
			Convey("success", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"rb",
					fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
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
		Convey("removeBucket", func() {
			patches := ApplyFunc(removeBucket, func(string) error {
				return fmt.Errorf("test removeBucket error")
			})
			defer patches.Reset()
			clearCmd()
			cmd := rootCmd
			args := []string{"rb",
				fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint, "-f"}
			cmd.SetArgs(args)
			e := cmd.Execute()
			fmt.Printf(" : %v", e)
			So(e, ShouldBeError)
		})
		Convey("removeBucket newClient", func() {
			patches := ApplyFunc(util.NewClient, func(config *util.Config, param *util.Param, bucketName string) (client *cos.Client, err error) {
				return nil, fmt.Errorf("test NewClient error")
			})
			defer patches.Reset()
			e := removeBucket("")
			fmt.Printf(" : %v", e)
			So(e, ShouldBeError)
		})
	})
}
