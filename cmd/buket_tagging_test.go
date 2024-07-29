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

func TestBucket_taggingCmd(t *testing.T) {
	fmt.Println("TestBucket_taggingCmd")
	testBucket = randStr(8)
	testAlias = testBucket + "-alias"
	setUp(testBucket, testAlias, testEndpoint, false)
	defer tearDown(testBucket, testAlias, testEndpoint)
	clearCmd()
	cmd := rootCmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	Convey("test coscli bucket_tagging", t, func() {
		Convey("success", func() {
			Convey("put", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"bucket-tagging", "--method", "put",
					fmt.Sprintf("cos://%s", testAlias), "testkey#testval"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("get", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"bucket-tagging", "--method", "get",
					fmt.Sprintf("cos://%s", testAlias)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("delete", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"bucket-tagging", "--method", "delete",
					fmt.Sprintf("cos://%s", testAlias)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})
		Convey("fail", func() {
			Convey("put", func() {
				Convey("not enough arguments", func() {
					clearCmd()
					cmd := rootCmd
					args := []string{"bucket-tagging", "--method", "put",
						fmt.Sprintf("cos://%s", testAlias)}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("clinet err", func() {
					clearCmd()
					cmd := rootCmd
					patches := ApplyFunc(util.NewClient, func(config *util.Config, param *util.Param, bucketName string) (client *cos.Client, err error) {
						return nil, fmt.Errorf("test put client error")
					})
					defer patches.Reset()
					args := []string{"bucket-tagging", "--method", "put",
						fmt.Sprintf("cos://%s", testAlias), "testval"}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("invalid tag", func() {
					clearCmd()
					cmd := rootCmd
					args := []string{"bucket-tagging", "--method", "put",
						fmt.Sprintf("cos://%s", testAlias), "testval"}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("PutTagging failed", func() {
					clearCmd()
					cmd := rootCmd
					args := []string{"bucket-tagging", "--method", "put",
						fmt.Sprintf("cos://%s", testAlias), "qcs:1#testval"}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
			})
			Convey("get", func() {
				Convey("not enough arguments", func() {
					clearCmd()
					cmd := rootCmd
					args := []string{"bucket-tagging", "--method", "get"}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("clinet err", func() {
					clearCmd()
					cmd := rootCmd
					patches := ApplyFunc(util.NewClient, func(config *util.Config, param *util.Param, bucketName string) (client *cos.Client, err error) {
						return nil, fmt.Errorf("test get client error")
					})
					defer patches.Reset()
					args := []string{"bucket-tagging", "--method", "get",
						fmt.Sprintf("cos://%s", testAlias)}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("get not exist", func() {
					clearCmd()
					cmd := rootCmd
					args := []string{"bucket-tagging", "--method", "get",
						fmt.Sprintf("cos://%s", testAlias)}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
			})
			Convey("delete", func() {
				Convey("not enough arguments", func() {
					clearCmd()
					cmd := rootCmd
					args := []string{"bucket-tagging", "--method", "delete"}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("delete bucket not exist", func() {
					clearCmd()
					cmd := rootCmd
					args := []string{"bucket-tagging", "--method", "delete",
						fmt.Sprintf("cos://%s", testAlias)}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeNil)
				})
				Convey("clinet err", func() {
					clearCmd()
					cmd := rootCmd
					patches := ApplyFunc(util.NewClient, func(config *util.Config, param *util.Param, bucketName string) (client *cos.Client, err error) {
						return nil, fmt.Errorf("test delete client error")
					})
					defer patches.Reset()
					args := []string{"bucket-tagging", "--method", "delete",
						fmt.Sprintf("cos://%s", testAlias)}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("DeleteTagging", func() {
					clearCmd()
					cmd := rootCmd
					var c *cos.BucketService
					patches := ApplyMethodFunc(reflect.TypeOf(c), "DeleteTagging", func(ctx context.Context) (*cos.Response, error) {
						return nil, fmt.Errorf("test delete tagging error")
					})
					defer patches.Reset()
					args := []string{"bucket-tagging", "--method", "delete",
						fmt.Sprintf("cos://%s", testAlias)}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
			})
		})
	})
}
