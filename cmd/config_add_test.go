package cmd

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConfigAddCmd(t *testing.T) {
	fmt.Println("TestConfigAddCmd")
	testBucket = randStr(8)
	setUp(testBucket, "", testEndpoint)
	defer tearDown(testBucket, "", testEndpoint)
	cmd := rootCmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	Convey("Test coscil config add", t, func() {
		// 成功不需要测，setup里就用过了
		// Convey("success", func() {
		// 	Convey("All have", func() {
		// 		cmd := rootCmd
		// 		args := []string{"config", "add", "-b",
		// 			fmt.Sprintf("%s-%s", testBucket, appID), "-e", testEndpoint, "-a", testAlias}
		// 		cmd.SetArgs(args)
		// 		e := cmd.Execute()
		// 		So(e, ShouldBeNil)
		// 	})
		// })
		Convey("fail", func() {
			Convey("Bucket already exist: name", func() {
				args := []string{"config", "add", "-b",
					fmt.Sprintf("%s-%s", testBucket, appID), "-e", testEndpoint, "-a", "testAlias"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %s", e.Error())
				So(e, ShouldBeError)
			})
			Convey("Bucket already exist: alias-name", func() {
				args := []string{"config", "add", "-b",
					fmt.Sprintf("%s-%s", "testBucket", appID), "-e", testEndpoint, "-a", fmt.Sprintf("%s-%s", testBucket, appID)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %s", e.Error())
				So(e, ShouldBeError)
			})
			Convey("Bucket already exist: alias", func() {
				args := []string{"config", "add", "-b",
					fmt.Sprintf("%s-%s", "testBucket", appID), "-e", testEndpoint, "-a", fmt.Sprintf("%s-%s", testBucket, appID)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %s", e.Error())
				So(e, ShouldBeError)
			})
		})
	})
}
