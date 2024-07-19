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
	setUp(testBucket, testAlias, testEndpoint)
	defer tearDown(testBucket, testAlias, testEndpoint)
	genDir(testDir, 3)
	defer delDir(testDir)
	localFileName := fmt.Sprintf("%s/small-file", testDir)
	cosFileName := fmt.Sprintf("cos://%s/%s", testAlias, "multi-small")
	cmd := rootCmd
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	args := []string{"cp", localFileName, cosFileName, "-r"}
	cmd.SetArgs(args)
	cmd.Execute()
	// 融合桶，无法临时创建
	Convey("Test coscli mv", t, func() {
		Convey("success", func() {
			Convey("ofs", func() {
				args := []string{"mv", fmt.Sprintf("cos://%s/pyjh.md", testOfsBucket), fmt.Sprintf("cos://%s/pyjh.md", testOfsBucket)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("not ofs", func() {
				args := []string{"mv", fmt.Sprintf("%s/0", cosFileName), fmt.Sprintf("cos://%s/%s/0", testAlias, "testmv")}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("not ofs but -r", func() {
				args := []string{"mv", cosFileName, fmt.Sprintf("cos://%s/%s", testAlias, "testmv"), "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})
		Convey("fail", func() {
			Convey("not enough arguments", func() {
				args := []string{"mv", "abc"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %s", e.Error())
				So(e, ShouldBeError)
			})
			Convey("storage-class", func() {
				args := []string{"mv", "cos://abc", "cos://abc", "--storage-class", "STANDARD"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %s", e.Error())
				So(e, ShouldBeError)
			})
			Convey("meta", func() {
				patches := ApplyFunc(util.MetaStringToHeader, func(string) (util.Meta, error) {
					var tmp util.Meta
					return tmp, fmt.Errorf("test meta error")
				})
				defer patches.Reset()
				args := []string{"mv", "cos://abc", "cos://abc", "--storage-class", ""}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %s", e.Error())
				So(e, ShouldBeError)
			})
			Convey("not cospath", func() {
				args := []string{"mv", "~/.abc", "cos://abc"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %s", e.Error())
				So(e, ShouldBeError)
			})
			Convey("not equal cospath", func() {
				args := []string{"mv", "cos://bcd", "cos://abc"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %s", e.Error())
				So(e, ShouldBeError)
			})
		})
	})
}
