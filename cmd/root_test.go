package cmd

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSkipCmd(t *testing.T) {
	fmt.Println("TestSkipCmd")
	testBucket = randStr(8)
	testAlias = testBucket + "-alias"
	setUp(testBucket, testAlias, testEndpoint, false, false)
	defer tearDown(testBucket, testAlias, testEndpoint, false)
	clearCmd()
	cmd := rootCmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	Convey("success", t, func() {
		cmd := rootCmd
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		cosFileName := fmt.Sprintf("cos://%s", testAlias)
		args := []string{"ls", cosFileName, "--init-skip", "-i", "123", "-k", "456"}
		cmd.SetArgs(args)
		e := cmd.Execute()
		fmt.Printf(" : %v", e)
		So(e, ShouldBeError)
	})
}
