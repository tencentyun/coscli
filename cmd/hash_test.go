package cmd

import (
	"coscli/util"
	"fmt"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestHashCmd(t *testing.T) {
	fmt.Println("TestHashCmd")
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
	Convey("Test coscli hash", t, func() {
		Convey("local file", func() {
			Convey("crc64", func() {
				clearCmd()
				cmd := rootCmd
				args = []string{"hash", fmt.Sprintf("%s/0", localFileName)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("md5", func() {
				clearCmd()
				cmd := rootCmd
				args = []string{"hash", fmt.Sprintf("%s/0", localFileName), "--type=md5"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})
		Convey("cos file", func() {
			Convey("crc64", func() {
				clearCmd()
				cmd := rootCmd
				args = []string{"hash", fmt.Sprintf("%s/0", cosFileName)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("md5", func() {
				clearCmd()
				cmd := rootCmd
				args = []string{"hash", fmt.Sprintf("%s/0", cosFileName), "--type=md5"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})
		Convey("fail", func() {
			Convey("New Client", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(util.NewClient, func(config *util.Config, param *util.Param, bucketName string) (client *cos.Client, err error) {
					return nil, fmt.Errorf("test client error")
				})
				defer patches.Reset()
				args = []string{"hash", fmt.Sprintf("%s/0", cosFileName)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("type error", func() {
				Convey("local file", func() {
					clearCmd()
					cmd := rootCmd
					args = []string{"hash", fmt.Sprintf("%s/0", localFileName), "--type=invalid"}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("cos file", func() {
					clearCmd()
					cmd := rootCmd
					args = []string{"hash", fmt.Sprintf("%s/0", cosFileName), "--type=invalid"}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
			})
		})
	})
}
