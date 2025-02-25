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

func TestLsCmd(t *testing.T) {
	fmt.Println("TestLsCmd")
	testBucket = randStr(8)
	testAlias = testBucket + "-alias"
	testOfsBucket = randStr(8)
	testOfsBucketAlias = testOfsBucket + "-alias"
	testVersionBucket = randStr(8)
	testVersionBucketAlias = testVersionBucket + "-alias"
	setUp(testBucket, testAlias, testEndpoint, false, false)
	defer tearDown(testBucket, testAlias, testEndpoint, false)
	setUp(testOfsBucket, testOfsBucketAlias, testEndpoint, true, false)
	defer tearDown(testOfsBucket, testOfsBucketAlias, testEndpoint, false)
	setUp(testVersionBucket, testVersionBucketAlias, testEndpoint, false, true)
	defer tearDown(testVersionBucket, testVersionBucketAlias, testEndpoint, true)
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

	ofsFileName := fmt.Sprintf("cos://%s/%s", testOfsBucketAlias, "multi-small")
	args = []string{"cp", localFileName, ofsFileName, "-r"}
	cmd.SetArgs(args)
	cmd.Execute()

	versioningFileName := fmt.Sprintf("cos://%s/%s", testVersionBucketAlias, "multi-small")
	args = []string{"cp", localFileName, versioningFileName, "-r"}
	cmd.SetArgs(args)
	cmd.Execute()

	Convey("Test coscli ls", t, func() {
		Convey("success", func() {
			Convey("无参数", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"ls"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("指定桶名", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"ls",
					fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint, "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("OFS", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"ls",
					fmt.Sprintf("cos://%s-%s", testOfsBucket, appID), "-e", testEndpoint, "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("多版本桶", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"ls",
					fmt.Sprintf("cos://%s-%s", testVersionBucket, appID), "-e", testEndpoint, "--all-versions", "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})
		Convey("fail", func() {
			Convey("参数--limit<0", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"ls", "--limit", "-1"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("FormatUrl", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(util.FormatUrl, func(urlStr string) (util.StorageUrl, error) {
					return nil, fmt.Errorf("test formaturl error")
				})
				defer patches.Reset()
				args := []string{"ls"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("New Client", func() {
				patches := ApplyFunc(util.NewClient, func(config *util.Config, param *util.Param, bucketName string) (client *cos.Client, err error) {
					return nil, fmt.Errorf("test new client error")
				})
				defer patches.Reset()
				Convey("no cosPath", func() {
					clearCmd()
					cmd := rootCmd
					args := []string{"ls"}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("cosPath", func() {
					clearCmd()
					cmd := rootCmd
					args := []string{"ls",
						fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
			})
			Convey("Head", func() {
				var c *cos.BucketService
				patches := ApplyMethodFunc(reflect.TypeOf(c), "Head", func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
					return nil, fmt.Errorf("test Head error")
				})
				defer patches.Reset()
				clearCmd()
				cmd := rootCmd
				args := []string{"ls",
					fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("ListObject", func() {
				patches := ApplyFunc(util.ListObjects, func(c *cos.Client, cosUrl util.StorageUrl, limit int, recursive bool, filters []util.FilterOptionType) error {
					return fmt.Errorf("test ListObject error")
				})
				defer patches.Reset()
				clearCmd()
				cmd := rootCmd
				args := []string{"ls",
					fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("not cos", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"ls",
					fmt.Sprintf("/%s-%s", testBucket, appID), "-e", testEndpoint}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("未开启多版本桶使用 --all-versions 参数", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"ls",
					fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint, "--all-versions"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("OFS桶使用 --all-versions 参数", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"ls",
					fmt.Sprintf("cos://%s-%s", testOfsBucket, appID), "-e", testEndpoint, "-r", "--all-versions"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
		})
	})
}
