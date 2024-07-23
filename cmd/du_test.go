package cmd

import (
	"coscli/util"
	"fmt"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestDuCmd(t *testing.T) {
	fmt.Println("TestDuCmd")
	testBucket = randStr(8)
	testAlias = testBucket + "-alias"
	setUp(testBucket, testAlias, testEndpoint, false)
	defer tearDown(testBucket, testAlias, testEndpoint)
	clearCmd()
	cmd := rootCmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	genDir(testDir, 3)
	defer delDir(testDir)
	localFileName := fmt.Sprintf("%s/small-file", testDir)
	cosFileName := fmt.Sprintf("cos://%s/%s", testAlias, "multi-small")
	args := []string{"cp", localFileName, cosFileName, "-r"}
	cmd.SetArgs(args)
	cmd.Execute()
	Convey("Test coscli du", t, func() {
		Convey("success", func() {
			Convey("duBucket", func() {
				clearCmd()
				cmd := rootCmd
				args = []string{"du", fmt.Sprintf("cos://%s", testAlias)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("duObjects", func() {
				clearCmd()
				cmd := rootCmd
				args = []string{"du", cosFileName}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})
		Convey("fail", func() {
			Convey("New Client", func() {
				patches := ApplyFunc(util.NewClient, func(config *util.Config, param *util.Param, bucketName string) (client *cos.Client, err error) {
					return nil, fmt.Errorf("test client error")
				})
				defer patches.Reset()
				Convey("duBucket", func() {
					clearCmd()
					cmd := rootCmd
					args = []string{"du", fmt.Sprintf("cos://%s", testAlias)}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("doObjects", func() {
					clearCmd()
					cmd := rootCmd
					args = []string{"du", cosFileName}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
			})
			Convey("GetObjectsListRecursive", func() {
				patches := ApplyFunc(util.GetObjectsListRecursive, func(c *cos.Client, prefix string, limit int, include string, exclude string, retryCount ...int) (objects []cos.Object, commonPrefixes []string, err error) {
					return nil, nil, fmt.Errorf("test getobjectlist error")
				})
				defer patches.Reset()
				Convey("duBucket", func() {
					clearCmd()
					cmd := rootCmd
					args = []string{"du", fmt.Sprintf("cos://%s", testAlias)}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("doObjects", func() {
					clearCmd()
					cmd := rootCmd
					args = []string{"du", cosFileName}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
			})
		})
	})
}
