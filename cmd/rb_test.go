package cmd

import (
	"fmt"
	"os/exec"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRbCmd(t *testing.T) {
	Convey("Test coscli rb", t, func() {
		Convey("success", func() {
			Convey("no force", func() {
				createTestBucket(testBucket, testEndpoint)
				cmd := exec.Command("../coscli", "rb",
					fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint)
				output, e := cmd.Output()
				fmt.Println(string(output))
				So(e, ShouldBeNil)
			})
			Convey("force", func() {
				createTestBucket(testBucket, testEndpoint)
				cmd := exec.Command("../coscli", "rb",
					fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint, "--force")
				output, e := cmd.Output()
				fmt.Println(string(output))
				So(e, ShouldBeNil)
			})
		})
		Convey("fail", func() {
			Convey("Invaild arguments", func() {
				cmd := exec.Command("../coscli", "rb")
				output, e := cmd.Output()
				fmt.Println(string(output))
				So(e, ShouldBeError)
			})
			Convey("Invalid bukcetIDName", func() {
				cmd := exec.Command("../coscli", "rb",
					fmt.Sprintf("cos:/"))
				output, e := cmd.Output()
				fmt.Println(string(output))
				So(e, ShouldBeError)
			})
			Convey("Not exist", func() {
				cmd := exec.Command("../coscli", "rb",
					fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint)
				output, e := cmd.Output()
				fmt.Println(string(output))
				So(e, ShouldBeError)
			})
		})
	})
}
