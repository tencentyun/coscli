package cmd

import (
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestLsCmd(t *testing.T) {
	fmt.Println("TestLsCmd")
	createTestBucket(testBucket, testEndpoint)
	defer deleteTestBucket(testBucket, testEndpoint)
	Convey("Test coscli ls", t, func() {
		Convey("ls bukect", func() {
			Convey("无参数", func() {
				cmd := rootCmd
				args := []string{"ls"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("参数--limit=0", func() {
				cmd := rootCmd
				args := []string{"ls", "--limit", "0"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			// Convey("参数--limit<0", func() {
			// 	cmd := exec.Command("../coscli", "ls", "--limit", "-1")
			// 	output, e := cmd.Output()
			// 	fmt.Println(string(output))
			// 	So(e, ShouldBeError)
			// })
		})
		Convey("ls object", func() {
			Convey("指定桶名", func() {
				cmd := rootCmd
				args := []string{"ls",
					fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})
	})
	time.Sleep(1 * time.Second)
}
