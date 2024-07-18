package cmd

import (
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConfigAddCmd(t *testing.T) {
	fmt.Println("TestConfigAddCmd")
	defer deleteConfig(testBucket)
	Convey("Test coscil config add", t, func() {
		Convey("success", func() {
			Convey("All have", func() {
				cmd := rootCmd
				args := []string{"config", "add", "-b",
					fmt.Sprintf("%s-%s", testBucket, appID), "-e", testEndpoint, "-a", testAlias}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})
		// Convey("fail", func() {
		// 	Convey("Bucket already exist: name", func() {
		// 		cmd := exec.Command("../coscli", "config", "add", "-b",
		// 			fmt.Sprintf("%s-%s", testBucket, appID), "-e", testEndpoint, "-a", testAlias1)
		// 		output, e := cmd.Output()
		// 		fmt.Println(string(output))
		// 		So(e, ShouldBeError)
		// 	})
		// 	Convey("Bucket already exist: alias-name", func() {
		// 		cmd := exec.Command("../coscli", "config", "add", "-b",
		// 			fmt.Sprintf("%s-%s", testBucket1, appID), "-e", testEndpoint1, "-a", fmt.Sprintf("%s-%s", testBucket, appID))
		// 		output, e := cmd.Output()
		// 		fmt.Println(string(output))
		// 		So(e, ShouldBeError)
		// 	})
		// 	Convey("Bucket already exist: alias", func() {
		// 		cmd := exec.Command("../coscli", "config", "add", "-b",
		// 			fmt.Sprintf("%s-%s", testBucket1, appID), "-e", testEndpoint1, "-a", testAlias)
		// 		output, e := cmd.Output()
		// 		fmt.Println(string(output))
		// 		So(e, ShouldBeError)
		// 	})
		// })
	})
	time.Sleep(1 * time.Second)
}
