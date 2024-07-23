package cmd

import (
	"context"
	"coscli/util"
	"fmt"
	"reflect"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestRestoreCmd(t *testing.T) {
	fmt.Println("TestRestoreCmd")
	testBucket = randStr(8)
	testAlias = testBucket + "-alias"
	setUp(testBucket, testAlias, testEndpoint, false)
	defer tearDown(testBucket, testAlias, testEndpoint)
	genDir(testDir, 3)
	defer delDir(testDir)
	localObject := fmt.Sprintf("%s/small-file/0", testDir)
	localFileName := fmt.Sprintf("%s/small-file", testDir)
	cosObject := fmt.Sprintf("cos://%s", testAlias)
	cosFileName := fmt.Sprintf("cos://%s/%s", testAlias, "multi-small")
	clearCmd()
	cmd := rootCmd
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	args1 := []string{"cp", localObject, cosObject, "--storage-class", "ARCHIVE"}
	args2 := []string{"cp", localFileName, cosFileName, "-r", "--storage-class", "ARCHIVE"}
	cmd.SetArgs(args1)
	cmd.Execute()
	cmd.SetArgs(args2)
	cmd.Execute()
	Convey("Test coscli restore", t, func() {
		Convey("object", func() {
			Convey("success", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"restore",
					fmt.Sprintf("%s/0", cosObject)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("Not enough arguments", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"restore"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("New Client", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(util.NewClient, func(config *util.Config, param *util.Param, bucketName string) (client *cos.Client, err error) {
					return nil, fmt.Errorf("test new client error")
				})
				defer patches.Reset()
				args := []string{"restore",
					fmt.Sprintf("%s/0", cosObject)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("PostRestore", func() {
				clearCmd()
				cmd := rootCmd
				var c *cos.ObjectService
				patches := ApplyMethodFunc(reflect.TypeOf(c), "PostRestore", func(ctx context.Context, name string, opt *cos.ObjectRestoreOptions, id ...string) (*cos.Response, error) {
					return nil, fmt.Errorf("test postrestore fail")
				})
				defer patches.Reset()
				args := []string{"restore",
					fmt.Sprintf("%s/0", cosObject)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
		})
		Convey("objects", func() {
			Convey("success", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"restore", cosFileName, "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("New Client", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(util.NewClient, func(config *util.Config, param *util.Param, bucketName string) (client *cos.Client, err error) {
					return nil, fmt.Errorf("test new client error")
				})
				defer patches.Reset()
				args := []string{"restore", cosFileName, "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})

			Convey("GetObjectsListRecursive", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(util.GetObjectsListRecursive, func(c *cos.Client, prefix string, limit int, include string, exclude string, retryCount ...int) (objects []cos.Object, commonPrefixes []string, err error) {
					return nil, nil, fmt.Errorf("test GetObjectsListRecursive error")
				})
				defer patches.Reset()
				args := []string{"restore", cosFileName, "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
		})
	})
}
