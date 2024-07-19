package cmd

// import (
// 	"fmt"
// 	"testing"
// 	"time"

// 	. "github.com/smartystreets/goconvey/convey"
// )

// func TestHashCmd(t *testing.T) {
// 	fmt.Println("TestHashCmd")
// 	setUp(testBucket, testAlias, testEndpoint)
// 	defer tearDown(testBucket, testAlias, testEndpoint)
// 	genDir(testDir, 3)
// 	defer delDir(testDir)
// 	time.Sleep(2 * time.Second)
// 	localFileName := fmt.Sprintf("%s/small-file", testDir)
// 	cosFileName := fmt.Sprintf("cos://%s/%s", testAlias, "multi-small")
// 	cmd := rootCmd
// 	args := []string{"cp", localFileName, cosFileName, "-r"}
// 	cmd.SetArgs(args)
// 	cmd.Execute()
// 	time.Sleep(1 * time.Second)
// 	Convey("Test coscli hash", t, func() {
// 		Convey("local file", func() {
// 			Convey("crc64", func() {
// 				args = []string{"hash", fmt.Sprintf("%s/0", localFileName)}
// 				cmd.SetArgs(args)
// 				e := cmd.Execute()
// 				So(e, ShouldBeNil)
// 			})
// 			Convey("md5", func() {
// 				args = []string{"hash", fmt.Sprintf("%s/0", localFileName), "--type=md5"}
// 				cmd.SetArgs(args)
// 				e := cmd.Execute()
// 				So(e, ShouldBeNil)
// 			})
// 		})
// 		Convey("cos file", func() {
// 			Convey("crc64", func() {
// 				args = []string{"hash", fmt.Sprintf("%s/0", cosFileName), "--type=crc64"}
// 				cmd.SetArgs(args)
// 				e := cmd.Execute()
// 				So(e, ShouldBeNil)
// 			})
// 			Convey("md5", func() {
// 				args = []string{"hash", fmt.Sprintf("%s/0", cosFileName), "--type=md5"}
// 				cmd.SetArgs(args)
// 				e := cmd.Execute()
// 				So(e, ShouldBeNil)
// 			})
// 		})
// 	})
// 	time.Sleep(1 * time.Second)
// }
