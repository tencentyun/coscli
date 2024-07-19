package cmd

// import (
// 	"fmt"
// 	"testing"
// 	"time"

// 	. "github.com/smartystreets/goconvey/convey"
// )

// func TestLspartsCmd(t *testing.T) {
// 	fmt.Println("TestLspartsCmd")
// 	setUp(testBucket, testAlias, testEndpoint)
// 	defer tearDown(testBucket, testAlias, testEndpoint)
// 	Convey("Test coscli lsparts", t, func() {
// 		Convey("success", func() {
// 			cmd := rootCmd
// 			args := []string{"lsparts", fmt.Sprintf("cos://%s", testAlias)}
// 			cmd.SetArgs(args)
// 			e := cmd.Execute()
// 			So(e, ShouldBeNil)
// 		})
// 	})
// 	time.Sleep(1 * time.Second)
// }
