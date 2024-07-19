package cmd

// import (
// 	"fmt"
// 	"testing"
// 	"time"

// 	. "github.com/smartystreets/goconvey/convey"
// )

// func TestSignurlCmd(t *testing.T) {
// 	fmt.Println("TestSignurlCmd")
// 	setUp(testBucket, testAlias, testEndpoint)
// 	defer tearDown(testBucket, testAlias, testEndpoint)
// 	genDir(testDir, 3)
// 	defer delDir(testDir)
// 	time.Sleep(2 * time.Second)
// 	localFileName := fmt.Sprintf("%s/small-file/0", testDir)
// 	cosFileName := fmt.Sprintf("cos://%s", testAlias)
// 	cmd := rootCmd
// 	cmd.SilenceUsage = true
// 	cmd.SilenceErrors = true
// 	args := []string{"cp", localFileName, cosFileName}
// 	cmd.SetArgs(args)
// 	cmd.Execute()
// 	time.Sleep(1 * time.Second)
// 	Convey("Test coscli signurl", t, func() {
// 		Convey("success", func() {
// 			cmd := rootCmd
// 			args := []string{"signurl",
// 				fmt.Sprintf("cos://%s/0", testAlias)}
// 			cmd.SetArgs(args)
// 			e := cmd.Execute()
// 			So(e, ShouldBeNil)
// 		})
// 		// Convey("failed", func() {
// 		// 	cmd := exec.Command("../coscli", "abort")
// 		// 	output, e := cmd.Output()
// 		// 	fmt.Println(string(output))
// 		// 	So(e, ShouldBeError)
// 		// })
// 	})
// 	time.Sleep(1 * time.Second)
// }
