package cmd

import (
	"context"
	"coscli/util"
	"fmt"
	"net/url"
	"reflect"
	"testing"
	"time"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestSignurlCmd(t *testing.T) {
	fmt.Println("TestSignurlCmd")
	testBucket = randStr(8)
	testAlias = testBucket + "-alias"
	setUp(testBucket, testAlias, testEndpoint, false)
	defer tearDown(testBucket, testAlias, testEndpoint)
	genDir(testDir, 3)
	defer delDir(testDir)
	localFileName := fmt.Sprintf("%s/small-file/0", testDir)
	cosFileName := fmt.Sprintf("cos://%s", testAlias)
	clearCmd()
	cmd := rootCmd
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	args := []string{"cp", localFileName, cosFileName}
	cmd.SetArgs(args)
	cmd.Execute()
	Convey("Test coscli signurl", t, func() {
		Convey("success", func() {
			clearCmd()
			cmd := rootCmd
			args := []string{"signurl",
				fmt.Sprintf("cos://%s/0", testAlias)}
			cmd.SetArgs(args)
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})
		Convey("failed", func() {
			Convey("Not enough arguments", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"abort"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("not cos", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"signurl",
					fmt.Sprintf("co//%s/0", testAlias)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("New Client", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(util.NewClient, func(config *util.Config, param *util.Param, bucketName string) (client *cos.Client, err error) {
					return nil, fmt.Errorf("test new client error")
				})
				defer patches.Reset()
				args := []string{"signurl",
					fmt.Sprintf("cos://%s/0", testAlias)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("Cover arguments and GetPresignedURL", func() {
				clearCmd()
				cmd := rootCmd
				var c *cos.ObjectService
				patches := ApplyMethodFunc(reflect.TypeOf(c), "GetPresignedURL", func(ctx context.Context, httpMethod string, name string, ak string, sk string, expired time.Duration, opt interface{}, signHost ...bool) (*url.URL, error) {
					return nil, fmt.Errorf("test getpresignedurl error")
				})
				defer patches.Reset()
				patches.ApplyFunc(util.NewClient, func(config *util.Config, param *util.Param, bucketName string) (client *cos.Client, err error) {
					return &cos.Client{}, nil
				})
				args := []string{"signurl",
					fmt.Sprintf("cos://%s/0", testAlias), "-i", "123", "-k", "123", "--token", "123"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
		})
	})
}
