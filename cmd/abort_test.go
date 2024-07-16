package cmd

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAbortCmd(t *testing.T) {
	fmt.Println("TestAbortCmd")
	createTestBucket(testBucket, testEndpoint)
	defer deleteTestBucket(testBucket, testEndpoint)
	Convey("Test coscli abort", t, func() {
		Convey("success", func() {
			cmd := rootCmd
			args := []string{"abort",
				fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint}
			cmd.SetArgs(args)
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})
		// Convey("failed", func() {
		// 	cmd := exec.Command("../coscli", "abort")
		// 	output, e := cmd.Output()
		// 	fmt.Println(string(output))
		// 	So(e, ShouldBeError)
		// })
	})
}
