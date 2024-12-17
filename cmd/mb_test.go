package cmd

import (
	"context"
	"coscli/util"
	"fmt"
	"reflect"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestMbCmd(t *testing.T) {
	fmt.Println("TestMbCmd")
	testBucket = randStr(8)
	testAlias = testBucket + "-alias"
	setUp(testBucket, testAlias, testEndpoint, false, false)
	defer tearDown(testBucket, testAlias, testEndpoint, false)
	clearCmd()
	cmd := rootCmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	Convey("Test coscli mb", t, func() {
		Convey("fail", func() {
			Convey("Already exist", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"mb",
					fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("not enough arguments", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"mb"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("Invalid arguments", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"mb", "cos://"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("No Endpoint", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(util.CreateClient, func(config *util.Config, param *util.Param, bucketIDName string) (client *cos.Client, err error) {
					return nil, fmt.Errorf(param.Endpoint)
				})
				defer patches.Reset()
				args := []string{"mb",
					fmt.Sprintf("cos://%s-%s", testBucket, appID), "--region", "guangzhou"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("Create Client", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(util.CreateClient, func(config *util.Config, param *util.Param, bucketIDName string) (client *cos.Client, err error) {
					return nil, fmt.Errorf("test create client error")
				})
				defer patches.Reset()
				args := []string{"mb",
					fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("Bucket.Put", func() {
				clearCmd()
				cmd := rootCmd
				var c *cos.BucketService
				patches := ApplyMethodFunc(reflect.TypeOf(c), "Put", func(ctx context.Context, opt *cos.BucketPutOptions) (*cos.Response, error) {
					return nil, fmt.Errorf("test bucket put error")
				})
				defer patches.Reset()
				args := []string{"mb",
					fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
		})
	})
}
