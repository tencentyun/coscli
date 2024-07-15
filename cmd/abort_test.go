package cmd

import (
	"fmt"
	"os/exec"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAbortCmd(t *testing.T) {
	createTestBucket(testBucket, testEndpoint)
	defer deleteTestBucket(testBucket, testEndpoint)
	Convey("Test coscli abort", t, func() {
		Convey("success", func() {
			cmd := exec.Command("../coscli", "abort",
				fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint)
			output, e := cmd.Output()
			fmt.Println(string(output))
			So(e, ShouldBeNil)
		})
		Convey("failed", func() {
			cmd := exec.Command("../coscli", "abort")
			output, e := cmd.Output()
			fmt.Println(string(output))
			So(e, ShouldBeError)
		})
	})
}
