package cmd

import (
	"coscli/util"
	"fmt"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestSymLinkCmd(t *testing.T) {
	fmt.Println("TestSymLinkCmd")
	testBucket = randStr(8)
	testAlias = testBucket + "-alias"
	linkKey := randStr(5)
	setUp(testBucket, testAlias, testEndpoint, false, false)
	defer tearDown(testBucket, testAlias, testEndpoint, false)
	genDir(testDir, 3)
	defer delDir(testDir)
	localFileName := fmt.Sprintf("%s/small-file/0", testDir)
	cosFileName := fmt.Sprintf("cos://%s", testAlias)
	clearCmd()
	cmd := rootCmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	args := []string{"cp", localFileName, cosFileName, "-r"}
	clearCmd()
	cmd = rootCmd
	cmd.SetArgs(args)
	cmd.Execute()
	Convey("test coscli symlink", t, func() {
		Convey("fail", func() {
			Convey("Not enough argument", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"symlink"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("FormatUrl", func() {
				patches := ApplyFunc(util.FormatUrl, func(urlStr string) (util.StorageUrl, error) {
					return nil, fmt.Errorf("test FormatUrl error")
				})
				defer patches.Reset()
				clearCmd()
				cmd := rootCmd
				args := []string{"symlink", "--method", "create", fmt.Sprintf("cos://%s/0", testAlias)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("Not CosUrl", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"symlink", "--method", "create", testAlias}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("NewClient", func() {
				patches := ApplyFunc(util.NewClient, func(config *util.Config, param *util.Param, bucketName string) (client *cos.Client, err error) {
					return nil, fmt.Errorf("test NewClient error")
				})
				defer patches.Reset()
				clearCmd()
				cmd := rootCmd
				args := []string{"symlink", "--method", "create", fmt.Sprintf("cos://%s/0", testAlias)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("CreateSymlink", func() {
				patches := ApplyFunc(util.CreateSymlink, func(c *cos.Client, cosUrl util.StorageUrl, linkKey string) error {
					return fmt.Errorf("test CreateSymlink error")
				})
				defer patches.Reset()
				clearCmd()
				cmd := rootCmd
				args := []string{"symlink", "--method", "create", fmt.Sprintf("cos://%s/0", testAlias)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("GetSymlink", func() {
				patches := ApplyFunc(util.GetSymlink, func(c *cos.Client, linkKey string) (res string, err error) {
					return "", fmt.Errorf("test GetSymlink error")
				})
				defer patches.Reset()
				clearCmd()
				cmd := rootCmd
				args := []string{"symlink", "--method", "get", fmt.Sprintf("cos://%s/0", testAlias)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("Invalid argument", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"symlink", "--method", "invalid", fmt.Sprintf("cos://%s/0", testAlias)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
		})
		Convey("success", func() {
			Convey("create", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"symlink", "--method", "create", fmt.Sprintf("cos://%s/0", testAlias), "--link", linkKey}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("get", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"symlink", "--method", "get", fmt.Sprintf("cos://%s", testAlias), "--link", linkKey}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})
	})
}
