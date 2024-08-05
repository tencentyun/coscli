package cmd

import (
	"coscli/util"
	"fmt"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestCatCmd(t *testing.T) {
	fmt.Println("TestCatCmd")
	testBucket = randStr(8)
	testAlias = testBucket + "-alias"
	setUp(testBucket, testAlias, testEndpoint, false)
	defer tearDown(testBucket, testAlias, testEndpoint)
	clearCmd()
	cmd := rootCmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	Convey("test coscli cat", t, func() {
		Convey("fail", func() {
			Convey("Not enough argument", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"cat"}
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
				args := []string{"cat", fmt.Sprintf("cos://%s", testAlias)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("Not CosUrl", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"cat", testAlias}
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
				args := []string{"cat", fmt.Sprintf("cos://%s", testAlias)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("CatObject", func() {
				patches := ApplyFunc(util.CatObject, func(c *cos.Client, cosUrl util.StorageUrl) error {
					return fmt.Errorf("test CatObject error")
				})
				defer patches.Reset()
				clearCmd()
				cmd := rootCmd
				args := []string{"cat", fmt.Sprintf("cos://%s", testAlias)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
		})
		Convey("success", func() {
			clearCmd()
			cmd := rootCmd
			args := []string{"cat", fmt.Sprintf("cos://%s", testAlias)}
			cmd.SetArgs(args)
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})
	})
}
