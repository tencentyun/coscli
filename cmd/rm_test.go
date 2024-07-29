package cmd

import (
	"coscli/util"
	"fmt"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestRmCmd(t *testing.T) {
	cmd := rootCmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	Convey("Test coscli rm", t, func() {
		Convey("Not enough arguments", func() {
			clearCmd()
			cmd := rootCmd
			args := []string{"rm"}
			cmd.SetArgs(args)
			e := cmd.Execute()
			fmt.Printf(" : %v", e)
			So(e, ShouldBeError)
		})
		Convey("Invalid arguments", func() {
			clearCmd()
			cmd := rootCmd
			args := []string{"rm", "invaild"}
			cmd.SetArgs(args)
			e := cmd.Execute()
			fmt.Printf(" : %v", e)
			So(e, ShouldBeError)
		})
		Convey("getFilesAndDirs", func() {
			Convey("GetObjectsListIterator", func() {
				patches := ApplyFunc(util.GetObjectsListIterator, func(c *cos.Client, prefix string, marker string, include string, exclude string) (objects []cos.Object, isTruncated bool, nextMarker string, commonPrefixes []string, err error) {
					return nil, false, "", nil, fmt.Errorf("test GetObjectsListIterator error")
				})
				defer patches.Reset()
				_, e := getFilesAndDirs(nil, "", "", "", "")
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("success", func() {
				patches := ApplyFunc(util.GetObjectsListIterator, func(c *cos.Client, prefix string, marker string, include string, exclude string) (objects []cos.Object, isTruncated bool, nextMarker string, commonPrefixes []string, err error) {
					return []cos.Object{{
						Key: "123",
					}}, false, "", []string{}, nil
				})
				defer patches.Reset()
				_, e := getFilesAndDirs(nil, "123", "", "", "")
				So(e, ShouldBeNil)
			})
		})
	})
}
