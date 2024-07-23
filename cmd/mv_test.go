package cmd

import (
	"coscli/util"
	"fmt"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMvCmd(t *testing.T) {
	fmt.Println("TestMvCmd")
	testBucket = randStr(8)
	testAlias = testBucket + "-alias"
	testOfsBucket = randStr(8)
	testOfsBucketAlias = testOfsBucket + "-alias"
	setUp(testBucket, testAlias, testEndpoint, false)
	defer tearDown(testBucket, testAlias, testEndpoint)
	// setUp(testOfsBucket, testOfsBucketAlias, testEndpoint, true)
	// defer tearDown(testOfsBucket, testOfsBucketAlias, testEndpoint)
	genDir(testDir, 3)
	defer delDir(testDir)
	localFileName := fmt.Sprintf("%s/small-file", testDir)
	cosFileName := fmt.Sprintf("cos://%s/%s", testAlias, "multi-small")
	// ofsFileName := fmt.Sprintf("cos://%s/%s", testOfsBucketAlias, "multi-small")
	clearCmd()
	cmd := rootCmd
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	args := []string{"cp", localFileName, cosFileName, "-r"}
	cmd.SetArgs(args)
	cmd.Execute()
	// args = []string{"cp", localFileName, ofsFileName, "-r"}
	// clearCmd()
	// cmd = rootCmd
	// cmd.SetArgs(args)
	// cmd.Execute()
	Convey("Test coscli mv", t, func() {
		Convey("success", func() {
			// Convey("ofs", func() {
			// 	clearCmd()
			// 	cmd := rootCmd
			// 	args := []string{"mv", fmt.Sprintf("%s/0", ofsFileName), fmt.Sprintf("cos://%s/0", testOfsBucketAlias)}
			// 	cmd.SetArgs(args)
			// 	e := cmd.Execute()
			// 	So(e, ShouldBeNil)
			// })
			Convey("not ofs", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"mv", fmt.Sprintf("%s/0", cosFileName), fmt.Sprintf("cos://%s/%s/0", testAlias, "testmv")}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("not ofs but -r", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"mv", cosFileName, fmt.Sprintf("cos://%s/%s", testAlias, "testmv"), "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})
		Convey("fail", func() {
			Convey("not enough arguments", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"mv", "abc"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %s", e)
				So(e, ShouldBeError)
			})
			Convey("storage-class", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"mv", "cos://abc", "cos://abc", "--storage-class", "STANDARD"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %s", e)
				So(e, ShouldBeError)
			})
			Convey("meta", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(util.MetaStringToHeader, func(string) (util.Meta, error) {
					var tmp util.Meta
					return tmp, fmt.Errorf("test meta error")
				})
				defer patches.Reset()
				args := []string{"mv", "cos://abc", "cos://abc", "--storage-class", ""}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %s", e)
				So(e, ShouldBeError)
			})
			Convey("not cospath", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"mv", "~/.abc", "cos://abc"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %s", e)
				So(e, ShouldBeError)
			})
			Convey("not equal cospath", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"mv", "cos://bcd", "cos://abc"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %s", e)
				So(e, ShouldBeError)
			})
		})
	})
}
