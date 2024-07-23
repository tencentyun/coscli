package cmd

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCpCmd(t *testing.T) {
	fmt.Println("TestCpCmd")
	testBucket1 = randStr(8)
	testAlias1 = testBucket1 + "-alias"
	testBucket2 = randStr(8)
	testAlias2 = testBucket2 + "-alias"
	setUp(testBucket1, testAlias1, testEndpoint, false)
	defer tearDown(testBucket1, testAlias1, testEndpoint)
	setUp(testBucket2, testAlias2, testEndpoint, false)
	defer tearDown(testBucket2, testAlias2, testEndpoint)
	clearCmd()
	cmd := rootCmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	genDir(testDir, 3)
	defer delDir(testDir)

	Convey("Test coscli cp", t, func() {
		Convey("upload", func() {
			Convey("上传单个小文件", func() {
				clearCmd()
				cmd := rootCmd
				localFileName := fmt.Sprintf("%s/small-file/0", testDir)
				cosFileName := fmt.Sprintf("cos://%s/%s", testAlias1, "single-small")
				args := []string{"cp", localFileName, cosFileName}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("上传多个小文件", func() {
				clearCmd()
				cmd := rootCmd
				localFileName := fmt.Sprintf("%s/small-file", testDir)
				cosFileName := fmt.Sprintf("cos://%s/%s", testAlias1, "multi-small")
				args := []string{"cp", localFileName, cosFileName, "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("上传单个大文件", func() {
				clearCmd()
				cmd := rootCmd
				localFileName := fmt.Sprintf("%s/big-file/0", testDir)
				cosFileName := fmt.Sprintf("cos://%s/%s", testAlias1, "single-big")
				args := []string{"cp", localFileName, cosFileName}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("上传多个大文件", func() {
				clearCmd()
				cmd := rootCmd
				localFileName := fmt.Sprintf("%s/big-file", testDir)
				cosFileName := fmt.Sprintf("cos://%s/%s", testAlias1, "multi-big")
				args := []string{"cp", localFileName, cosFileName, "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})
		Convey("Copy", func() {
			Convey("桶内拷贝单个文件", func() {
				clearCmd()
				cmd := rootCmd
				srcPath := fmt.Sprintf("cos://%s/%s", testAlias1, "single-big")
				dstPath := fmt.Sprintf("cos://%s/%s", testAlias1, "single-copy")
				args := []string{"cp", srcPath, dstPath}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("桶内拷贝多个文件", func() {
				clearCmd()
				cmd := rootCmd
				srcPath := fmt.Sprintf("cos://%s/%s", testAlias1, "multi-big")
				dstPath := fmt.Sprintf("cos://%s/%s", testAlias1, "multi-copy")
				args := []string{"cp", srcPath, dstPath, "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("跨桶拷贝单个小文件", func() {
				clearCmd()
				cmd := rootCmd
				srcPath := fmt.Sprintf("cos://%s/%s", testAlias1, "single-small")
				dstPath := fmt.Sprintf("cos://%s/%s", testAlias2, "single-copy-small")
				args := []string{"cp", srcPath, dstPath}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("跨桶拷贝多个小文件", func() {
				clearCmd()
				cmd := rootCmd
				srcPath := fmt.Sprintf("cos://%s/%s", testAlias1, "multi-small")
				dstPath := fmt.Sprintf("cos://%s/%s", testAlias2, "multi-copy-small")
				args := []string{"cp", srcPath, dstPath, "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("跨桶拷贝单个大文件", func() {
				clearCmd()
				cmd := rootCmd
				srcPath := fmt.Sprintf("cos://%s/%s", testAlias1, "single-big")
				dstPath := fmt.Sprintf("cos://%s/%s", testAlias2, "single-copy-big")
				args := []string{"cp", srcPath, dstPath}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("跨桶拷贝多个大文件", func() {
				clearCmd()
				cmd := rootCmd
				srcPath := fmt.Sprintf("cos://%s/%s", testAlias1, "multi-big")
				dstPath := fmt.Sprintf("cos://%s/%s", testAlias2, "multi-copy-big")
				args := []string{"cp", srcPath, dstPath, "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})
		Convey("Download", func() {
			Convey("下载单个小文件", func() {
				clearCmd()
				cmd := rootCmd
				localFileName := fmt.Sprintf("%s/download/single-small", testDir)
				cosFileName := fmt.Sprintf("cos://%s/%s", testAlias2, "single-copy-small")
				args := []string{"cp", cosFileName, localFileName}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("下载多个小文件", func() {
				clearCmd()
				cmd := rootCmd
				localFileName := fmt.Sprintf("%s/download/small-file", testDir)
				cosFileName := fmt.Sprintf("cos://%s/%s", testAlias2, "multi-copy-small")
				args := []string{"cp", cosFileName, localFileName, "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("下载单个大文件", func() {
				clearCmd()
				cmd := rootCmd
				localFileName := fmt.Sprintf("%s/download/single-big", testDir)
				cosFileName := fmt.Sprintf("cos://%s/%s", testAlias2, "single-copy-big")
				args := []string{"cp", cosFileName, localFileName}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("下载多个大文件", func() {
				clearCmd()
				cmd := rootCmd
				localFileName := fmt.Sprintf("%s/download/big-file", testDir)
				cosFileName := fmt.Sprintf("cos://%s/%s", testAlias2, "multi-copy-big")
				args := []string{"cp", cosFileName, localFileName, "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})
	})

}
