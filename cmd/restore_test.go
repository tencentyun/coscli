package cmd

import (
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRestoreCmd(t *testing.T) {
	fmt.Println("TestRestoreCmd")
	setUp(testBucket, testAlias, testEndpoint)
	defer tearDown(testBucket, testAlias, testEndpoint)
	genDir(testDir, 3)
	defer delDir(testDir)
	time.Sleep(2 * time.Second)
	localObject := fmt.Sprintf("%s/small-file/0", testDir)
	localFileName := fmt.Sprintf("%s/small-file", testDir)
	cosObject := fmt.Sprintf("cos://%s", testAlias)
	cosFileName := fmt.Sprintf("cos://%s/%s", testAlias, "multi-small")
	cmd := rootCmd
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	args1 := []string{"cp", localObject, cosObject, "--storage-class", "ARCHIVE"}
	args2 := []string{"cp", localFileName, cosFileName, "-r", "--storage-class", "ARCHIVE"}
	cmd.SetArgs(args1)
	cmd.Execute()
	time.Sleep(1 * time.Second)
	cmd.SetArgs(args2)
	cmd.Execute()
	time.Sleep(1 * time.Second)
	Convey("Test coscli restore", t, func() {
		Convey("success", func() {
			Convey("object", func() {
				cmd := rootCmd
				args := []string{"restore",
					fmt.Sprintf("cos://%s/0", testAlias)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("objects", func() {
				cmd := rootCmd
				args := []string{"restore",
					fmt.Sprintf("cos://%s/multi-small", testAlias), "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})
		// Convey("failed", func() {
		// 	cmd := exec.Command("../coscli", "abort")
		// 	output, e := cmd.Output()
		// 	fmt.Println(string(output))
		// 	So(e, ShouldBeError)
		// })
	})
	time.Sleep(1 * time.Second)
}
