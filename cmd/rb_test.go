package cmd

import (
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRbCmd(t *testing.T) {
	fmt.Println("TestRbCmd")
	Convey("Test coscli rb", t, func() {
		Convey("success", func() {
			Convey("no force", func() {
				createTestBucket(testBucket, testEndpoint)
				cmd := rootCmd
				args := []string{"rb",
					fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("force", func() {
				createTestBucket(testBucket, testEndpoint)
				cmd := rootCmd
				args := []string{"rb",
					fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint, "--force"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})
		// Convey("fail", func() {
		// 	Convey("Invaild arguments", func() {
		// 		cmd := exec.Command("../coscli", "rb")
		// 		output, e := cmd.Output()
		// 		fmt.Println(string(output))
		// 		So(e, ShouldBeError)
		// 	})
		// 	Convey("Invalid bukcetIDName", func() {
		// 		cmd := exec.Command("../coscli", "rb",
		// 			fmt.Sprintf("cos:/"))
		// 		output, e := cmd.Output()
		// 		fmt.Println(string(output))
		// 		So(e, ShouldBeError)
		// 	})
		// 	Convey("Not exist", func() {
		// 		cmd := exec.Command("../coscli", "rb",
		// 			fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint)
		// 		output, e := cmd.Output()
		// 		fmt.Println(string(output))
		// 		So(e, ShouldBeError)
		// 	})
		// })
	})
	time.Sleep(1 * time.Second)
}
