package cmd

import (
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

//var testMbBucket string = "coscli-test-mb"

func TestMbCmd(t *testing.T) {
	fmt.Println("TestMbCmd")
	defer deleteTestBucket(testBucket, testEndpoint)
	Convey("Test coscli mb", t, func() {
		Convey("success", func() {
			cmd := rootCmd
			args := []string{"mb",
				fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint}
			cmd.SetArgs(args)
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})
		// Convey("fail", func() {
		// 	Convey("Already exist", func() {
		// 		cmd := exec.Command("../coscli", "mb",
		// 			fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint)
		// 		output, e := cmd.Output()
		// 		fmt.Println(string(output))
		// 		So(e, ShouldBeError)
		// 	})
		// 	Convey("Invalid arguments", func() {
		// 		cmd := exec.Command("../coscli", "mb",
		// 			fmt.Sprintf("cos://%s-%s", testBucket, appID))
		// 		output, e := cmd.Output()
		// 		fmt.Println(string(output))
		// 		So(e, ShouldBeError)
		// 	})
		// })
	})
	time.Sleep(1 * time.Second)
}
