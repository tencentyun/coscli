package cmd

import (
	"coscli/util"
	"fmt"
	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tencentyun/cos-go-sdk-v5"
	"testing"
)

func TestRmCmd(t *testing.T) {
	fmt.Println("TestRmCmd")
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
	Convey("Test coscli rm", t, func() {
		Convey("success", func() {
			Convey("rm single object", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(util.GetBucketVersioning, func(c *cos.Client) (res *cos.BucketGetVersionResult, resp *cos.Response, err error) {
					res = &cos.BucketGetVersionResult{
						Status: util.VersionStatusEnabled,
					}
					return res, nil, nil
				})
				defer patches.Reset()
				patches = ApplyFunc(util.CheckCosObjectExist, func(c *cos.Client, prefix string, id ...string) (exist bool, err error) {
					return true, nil
				})
				args := []string{"rm", versioningFileName, "--version-id", "123"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeNil)
			})
			Convey("rm versions", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"rm", versioningFileName, "--all-versions", "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeNil)
			})
			Convey("rm cos objects", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"rm", cosFileName, "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeNil)
			})
			Convey("rm ofs objects", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"rm", ofsFileName, "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeNil)
			})
		})
		Convey("fail", func() {
			Convey("Not enough arguments", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"rm"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("Invalid arguments", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"rm", "invaild"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("versionId use in recursive", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"rm", versioningFileName, "--version-id", "123", "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("all-versions use in single object", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"rm", versioningFileName, "--all-versions"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
		})

	})
}
