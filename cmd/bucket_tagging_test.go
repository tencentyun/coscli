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

func TestBucketTaggingCmd(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("test coscli bucket_tagging", t, func() {
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
			cmd.SetArgs([]string{"bucket-tagging", "--method", "get", "cos:/test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("put without tags", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "PutTagging",
				func(ctx context.Context, opt *cos.BucketPutTaggingOptions) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-tagging", "--method", "put", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("invalid method", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "GetTagging",
				func(ctx context.Context) (*cos.BucketGetTaggingResult, *cos.Response, error) {
					return &cos.BucketGetTaggingResult{}, &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-tagging", "--method", "invalid", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("put success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "PutTagging",
				func(ctx context.Context, opt *cos.BucketPutTaggingOptions) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-tagging", "--method", "put", "cos://test-alias", "tag1#test1", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("add success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "GetTagging",
				func(ctx context.Context) (*cos.BucketGetTaggingResult, *cos.Response, error) {
					return &cos.BucketGetTaggingResult{}, &cos.Response{}, nil
				})
			patches.ApplyMethodFunc(reflect.TypeOf(b), "PutTagging",
				func(ctx context.Context, opt *cos.BucketPutTaggingOptions) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-tagging", "--method", "add", "cos://test-alias", "tag3#test3", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("get success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "GetTagging",
				func(ctx context.Context) (*cos.BucketGetTaggingResult, *cos.Response, error) {
					return &cos.BucketGetTaggingResult{}, &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-tagging", "--method", "get", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("delete all success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "DeleteTagging",
				func(ctx context.Context) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-tagging", "--method", "delete", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("delete specific tags success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "GetTagging",
				func(ctx context.Context) (*cos.BucketGetTaggingResult, *cos.Response, error) {
					return &cos.BucketGetTaggingResult{
						TagSet: []cos.BucketTaggingTag{{Key: "tag1", Value: "test1"}},
					}, &cos.Response{}, nil
				})
			patches.ApplyMethodFunc(reflect.TypeOf(b), "PutTagging",
				func(ctx context.Context, opt *cos.BucketPutTaggingOptions) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-tagging", "--method", "delete", "cos://test-alias", "tag1#test1", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("put error", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "PutTagging",
				func(ctx context.Context, opt *cos.BucketPutTaggingOptions) (*cos.Response, error) {
					return nil, fmt.Errorf("test put tagging error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-tagging", "--method", "put", "cos://test-alias", "tag1#test1", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("get error", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "GetTagging",
				func(ctx context.Context) (*cos.BucketGetTaggingResult, *cos.Response, error) {
					return nil, nil, fmt.Errorf("test get tagging error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-tagging", "--method", "get", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("delete error", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "DeleteTagging",
				func(ctx context.Context) (*cos.Response, error) {
					return nil, fmt.Errorf("test delete tagging error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-tagging", "--method", "delete", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("NewClient error", func() {
			patches = ApplyFunc(util.NewClient, func(cfg *util.Config, param *util.Param, bucketName string) (*cos.Client, error) {
				return nil, fmt.Errorf("test NewClient error")
			})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-tagging", "--method", "get", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})
	})
}
