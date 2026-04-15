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

func TestBucketVersioningCmd(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("test coscli bucket_versioning", t, func() {
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
			cmd.SetArgs([]string{"bucket-versioning", "--method", "get", "cos:/test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("invalid method", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "GetVersioning",
				func(ctx context.Context) (*cos.BucketGetVersionResult, *cos.Response, error) {
					return &cos.BucketGetVersionResult{}, &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-versioning", "--method", "add", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("put success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "PutVersioning",
				func(ctx context.Context, opt *cos.BucketPutVersionOptions) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-versioning", "--method", "put", "cos://test-alias", "Enabled", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("get success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "GetVersioning",
				func(ctx context.Context) (*cos.BucketGetVersionResult, *cos.Response, error) {
					return &cos.BucketGetVersionResult{Status: "Enabled"}, &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-versioning", "--method", "get", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("put error", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "PutVersioning",
				func(ctx context.Context, opt *cos.BucketPutVersionOptions) (*cos.Response, error) {
					return nil, fmt.Errorf("test put versioning error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-versioning", "--method", "put", "cos://test-alias", "Enabled", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("get error", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "GetVersioning",
				func(ctx context.Context) (*cos.BucketGetVersionResult, *cos.Response, error) {
					return nil, nil, fmt.Errorf("test get versioning error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-versioning", "--method", "get", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("invalid cos url format", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-versioning", "--method", "get", "cos:///invalid-object", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("NewClient error", func() {
			patches = ApplyFunc(util.NewClient, func(cfg *util.Config, param *util.Param, bucketName string) (*cos.Client, error) {
				return nil, fmt.Errorf("test NewClient error")
			})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-versioning", "--method", "get", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("put not enough args", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "PutVersioning",
				func(ctx context.Context, opt *cos.BucketPutVersionOptions) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-versioning", "--method", "put", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("put invalid status", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "PutVersioning",
				func(ctx context.Context, opt *cos.BucketPutVersionOptions) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-versioning", "--method", "put", "cos://test-alias", "InvalidStatus", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("get closed status", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "GetVersioning",
				func(ctx context.Context) (*cos.BucketGetVersionResult, *cos.Response, error) {
					return &cos.BucketGetVersionResult{Status: ""}, &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-versioning", "--method", "get", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})
	})
}
