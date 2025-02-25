package cmd

import (
	"coscli/util"
	"fmt"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestRestoreCmd(t *testing.T) {
	fmt.Println("TestRestoreCmd")
	testBucket = randStr(8)
	testAlias = testBucket + "-alias"
	setUp(testBucket, testAlias, testEndpoint, false, false)
	defer tearDown(testBucket, testAlias, testEndpoint, false)
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
	clearCmd()
	cmd = rootCmd
	cmd.SetArgs(args2)
	cmd.Execute()
	Convey("Test coscli restore", t, func() {
		Convey("success", func() {
			Convey("RestoreObject", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"restore",
					fmt.Sprintf("%s/0", cosObject)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("RestoreObjects", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"restore", cosFileName, "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})
		Convey("fail", func() {
			Convey("Not enough arguments", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"restore"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("days over range", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"restore", fmt.Sprintf("%s/0", cosObject), "--days", "366"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("FormatUrl", func() {
				patches := ApplyFunc(util.FormatUrl, func(urlStr string) (util.StorageUrl, error) {
					return nil, fmt.Errorf("test formaturl fail")
				})
				defer patches.Reset()
				clearCmd()
				cmd := rootCmd
				args := []string{"restore", "invalid"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("not cos url", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"restore", "invalid"}
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
				args := []string{"restore", cosFileName, "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})

			Convey("RestoreObjects", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(util.RestoreObjects, func(c *cos.Client, cosUrl util.StorageUrl, fo *util.FileOperations) error {
					return fmt.Errorf("test RestoreObjects error")
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
