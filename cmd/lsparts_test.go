package cmd

import (
	"coscli/util"
	"fmt"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestLspartsCmd(t *testing.T) {
	fmt.Println("TestLspartsCmd")
	testBucket = randStr(8)
	testAlias = testBucket + "-alias"
	setUp(testBucket, testAlias, testEndpoint, false)
	defer tearDown(testBucket, testAlias, testEndpoint)
	clearCmd()
	cmd := rootCmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	Convey("Test coscli lsparts", t, func() {
		Convey("success", func() {
			clearCmd()
			cmd := rootCmd
			args := []string{"lsparts", fmt.Sprintf("cos://%s", testAlias)}
			cmd.SetArgs(args)
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})
		Convey("fail", func() {
			Convey("limit invalid", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"lsparts", fmt.Sprintf("cos://%s", testAlias), "--limit", "-1"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("New Client", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(util.NewClient, func(config *util.Config, param *util.Param, bucketName string) (client *cos.Client, err error) {
					return nil, fmt.Errorf("test formaturl error")
				})
				defer patches.Reset()
				args := []string{"lsparts", fmt.Sprintf("cos://%s", testAlias)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("GetUploadsListRecursive", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(util.GetUploadsListRecursive, func(c *cos.Client, prefix string, limit int, include string, exclude string) (uploads []util.UploadInfo, err error) {
					return nil, fmt.Errorf("test getuploadslist error")
				})
				defer patches.Reset()
				args := []string{"lsparts", fmt.Sprintf("cos://%s", testAlias)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
		})
	})
}
