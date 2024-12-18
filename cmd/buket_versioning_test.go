package cmd

import (
	"coscli/util"
	"fmt"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestBucketVersionCmd(t *testing.T) {
	fmt.Println("TestBucketVersionCmd")
	testBucket = randStr(8)
	testAlias = testBucket + "-alias"
	setUp(testBucket, testAlias, testEndpoint, false, false)
	defer tearDown(testBucket, testAlias, testEndpoint, false)
	clearCmd()
	cmd := rootCmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	Convey("test coscli bucket_versioning", t, func() {
		Convey("success", func() {
			Convey("put", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"bucket-versioning", "--method", "put",
					fmt.Sprintf("cos://%s", testAlias), "Enabled"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("get", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"bucket-versioning", "--method", "get",
					fmt.Sprintf("cos://%s", testAlias)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("get status closed", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(util.GetBucketVersioning, func(c *cos.Client) (res *cos.BucketGetVersionResult, resp *cos.Response, err error) {
					return &cos.BucketGetVersionResult{}, nil, nil
				})
				defer patches.Reset()
				args := []string{"bucket-versioning", "--method", "get",
					fmt.Sprintf("cos://%s", testAlias)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})
		Convey("fail", func() {
			Convey("cos url error", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"bucket-versioning", "--method", "put",
					fmt.Sprintf("co:/%s", testAlias), "Enabled"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("cos url format error", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(util.FormatUrl, func(urlStr string) (util.StorageUrl, error) {
					return nil, fmt.Errorf("cos url format error")
				})
				defer patches.Reset()
				args := []string{"bucket-versioning", "--method", "put",
					fmt.Sprintf("cos://%s", testAlias), "Enabled"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("put", func() {
				Convey("not enough arguments", func() {
					clearCmd()
					cmd := rootCmd
					args := []string{"bucket-versioning", "--method", "put",
						fmt.Sprintf("cos://%s", testAlias)}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("clinet err", func() {
					clearCmd()
					cmd := rootCmd
					patches := ApplyFunc(util.NewClient, func(config *util.Config, param *util.Param, bucketName string) (client *cos.Client, err error) {
						return nil, fmt.Errorf("test put client error")
					})
					defer patches.Reset()
					args := []string{"bucket-versioning", "--method", "put",
						fmt.Sprintf("cos://%s", testAlias), "Enabled"}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("invalid status", func() {
					clearCmd()
					cmd := rootCmd
					args := []string{"bucket-versioning", "--method", "put",
						fmt.Sprintf("cos://%s", testAlias), "testStatus"}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("put bucket versioning error", func() {
					clearCmd()
					cmd := rootCmd
					patches := ApplyFunc(util.PutBucketVersioning, func(c *cos.Client, status string) (res *cos.Response, err error) {
						return nil, fmt.Errorf("put bucket versioning error")
					})
					defer patches.Reset()
					args := []string{"bucket-versioning", "--method", "put",
						fmt.Sprintf("cos://%s", testAlias), "Enabled"}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
			})
			Convey("get", func() {
				Convey("clinet err", func() {
					clearCmd()
					cmd := rootCmd
					patches := ApplyFunc(util.NewClient, func(config *util.Config, param *util.Param, bucketName string) (client *cos.Client, err error) {
						return nil, fmt.Errorf("test get client error")
					})
					defer patches.Reset()
					args := []string{"bucket-versioning", "--method", "get",
						fmt.Sprintf("cos://%s", testAlias)}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("get bucket versioning error", func() {
					clearCmd()
					cmd := rootCmd
					patches := ApplyFunc(util.GetBucketVersioning, func(c *cos.Client) (res *cos.BucketGetVersionResult, resp *cos.Response, err error) {
						return nil, nil, fmt.Errorf("get bucket versioning error")
					})
					defer patches.Reset()
					args := []string{"bucket-versioning", "--method", "get",
						fmt.Sprintf("cos://%s", testAlias)}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
			})
		})
	})
}
