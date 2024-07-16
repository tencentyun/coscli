package cmd

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestDuCmd(t *testing.T) {
	fmt.Println("TestDuCmd")
	setUp(testBucket, testAlias, testEndpoint)
	defer tearDown(testBucket, testAlias, testEndpoint)
	genDir(testDir, 3)
	defer delDir(testDir)
	Convey("Test coscli du", t, func() {
		localFileName := fmt.Sprintf("%s/small-file", testDir)
		cosFileName := fmt.Sprintf("cos://%s/%s", testAlias, "multi-small")
		cmd := rootCmd
		args := []string{"cp", localFileName, cosFileName, "-r"}
		cmd.SetArgs(args)
		cmd.Execute()
		Convey("duBucket", func() {
			args = []string{"du", fmt.Sprintf("cos://%s", testAlias)}
			cmd.SetArgs(args)
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})
		Convey("duObjects", func() {
			args = []string{"du", cosFileName}
			cmd.SetArgs(args)
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})
	})
}
