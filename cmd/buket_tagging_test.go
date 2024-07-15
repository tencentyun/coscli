package cmd

import (
	"fmt"
	"os/exec"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestBucket_taggingCmd(t *testing.T) {
	setUp(testBucket, testAlias, testEndpoint)
	defer tearDown(testBucket, testAlias, testEndpoint)
	Convey("test coscli bucket_tagging", t, func() {
		Convey("success", func() {
			Convey("put", func() {
				cmd := exec.Command("../coscli", "bucket-tagging", "--method", "put",
					fmt.Sprintf("cos://%s", testAlias), "testkey#testval")
				output, e := cmd.Output()
				fmt.Println(string(output))
				So(e, ShouldBeNil)
			})
			Convey("get", func() {
				cmd := exec.Command("../coscli", "bucket-tagging", "--method", "get",
					fmt.Sprintf("cos://%s", testAlias))
				output, e := cmd.Output()
				fmt.Println(string(output))
				So(e, ShouldBeNil)
			})
			Convey("delete", func() {
				cmd := exec.Command("../coscli", "bucket-tagging", "--method", "delete",
					fmt.Sprintf("cos://%s", testAlias))
				output, e := cmd.Output()
				fmt.Println(string(output))
				So(e, ShouldBeNil)
			})
		})
		Convey("fail", func() {
			Convey("put", func() {
				Convey("not enough arguments", func() {
					cmd := exec.Command("../coscli", "bucket-tagging", "--method", "put",
						fmt.Sprintf("cos://%s", testAlias))
					output, e := cmd.Output()
					fmt.Println(string(output))
					So(e, ShouldBeError)
				})
				Convey("invalid tag", func() {
					cmd := exec.Command("../coscli", "bucket-tagging", "--method", "put",
						fmt.Sprintf("cos://%s", testAlias), "testval")
					output, e := cmd.Output()
					fmt.Println(string(output))
					So(e, ShouldBeError)
				})
				Convey("PutTagging failed", func() {
					cmd := exec.Command("../coscli", "bucket-tagging", "--method", "put",
						fmt.Sprintf("cos://%s", testAlias), "qcs:1#testval")
					output, e := cmd.Output()
					fmt.Println(string(output))
					So(e, ShouldBeError)
				})
			})
			Convey("get", func() {
				Convey("not enough arguments", func() {
					cmd := exec.Command("../coscli", "bucket-tagging", "--method", "get")
					output, e := cmd.Output()
					fmt.Println(string(output))
					So(e, ShouldBeError)
				})
			})
			Convey("delete", func() {
				Convey("not enough arguments", func() {
					cmd := exec.Command("../coscli", "bucket-tagging", "--method", "delete")
					output, e := cmd.Output()
					fmt.Println(string(output))
					So(e, ShouldBeError)
				})
			})
		})
	})
}
