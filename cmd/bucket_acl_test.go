package cmd

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tencentyun/cos-go-sdk-v5"

	"coscli/util"
)

func TestBucketAclCmd(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("test coscli bucket_acl", t, func() {
		var patches *Patches
		Reset(func() {
			if patches != nil {
				patches.Reset()
				patches = nil
			}
			clearCmd()
		})

		Convey("cos path error", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-acl", "--method", "put",
				"cos:/test-alias", "--grant-read", `id="100000000003",id="100000000002"`, "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("put success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "PutACL",
				func(ctx context.Context, opt *cos.BucketPutACLOptions) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-acl", "--method", "put",
				"cos://test-alias", "--grant-read", `id="100000000003",id="100000000002"`, "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("get success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "GetACL",
				func(ctx context.Context) (*cos.BucketGetACLResult, *cos.Response, error) {
					return &cos.BucketGetACLResult{}, &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-acl", "--method", "get", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("invalid method", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "PutACL",
				func(ctx context.Context, opt *cos.BucketPutACLOptions) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-acl", "--method", "add",
				"cos://test-alias", "--grant-read", `id="100000000003",id="100000000002"`, "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("put acl error", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "PutACL",
				func(ctx context.Context, opt *cos.BucketPutACLOptions) (*cos.Response, error) {
					return nil, fmt.Errorf("test put acl error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-acl", "--method", "put",
				"cos://test-alias", "--grant-read", `id="100000000003"`, "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("get acl error", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "GetACL",
				func(ctx context.Context) (*cos.BucketGetACLResult, *cos.Response, error) {
					return nil, nil, fmt.Errorf("test get acl error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-acl", "--method", "get", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("NewClient error", func() {
			patches = ApplyFunc(util.NewClient, func(cfg *util.Config, param *util.Param, bucketName string) (*cos.Client, error) {
				return nil, fmt.Errorf("test NewClient error")
			})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-acl", "--method", "get", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})
	})
}
