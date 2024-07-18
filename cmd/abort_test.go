package cmd

import (
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAbortCmd(t *testing.T) {
	fmt.Println("TestAbortCmd")
	createTestBucket(testBucket, testEndpoint)
	defer deleteTestBucket(testBucket, testEndpoint)
	cmd := rootCmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	Convey("Test coscli abort", t, func() {
		Convey("success", func() {

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
	time.Sleep(1 * time.Second)
}
