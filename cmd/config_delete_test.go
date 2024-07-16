package cmd

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConfigDeleteCmd(t *testing.T) {
	fmt.Println("TestConfigDeleteCmd")
	addConfig(testBucket, testEndpoint)
	Convey("Test coscil config delete", t, func() {
		Convey("success", func() {
			cmd := rootCmd
			args := []string{"config", "delete", "-a",
				fmt.Sprintf("%s-%s", testBucket, appID)}
			cmd.SetArgs(args)
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})
		// Convey("fail", func() {
		// 	cmd := exec.Command("../coscli", "config", "delete", "-a", testAlias)
		// 	output, e := cmd.Output()
		// 	fmt.Println(string(output))
		// 	So(e, ShouldBeError)
		// })
	})
}
