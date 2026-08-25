package cmd

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tencentyun/cos-go-sdk-v5"

	"coscli/util"
)

func TestRbCmd(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("Test coscli rb", t, func() {
		var patches *Patches
		Reset(func() {
			if patches != nil {
				patches.Reset()
				patches = nil
			}
			clearCmd()
		})

		Convey("invalid arguments", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"rb", "cos://test-alias/object", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("delete bucket success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Delete",
				func(ctx context.Context, opt ...*cos.BucketDeleteOptions) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"rb", "cos://test-bucket-1234567890", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("delete bucket error", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Delete",
				func(ctx context.Context, opt ...*cos.BucketDeleteOptions) (*cos.Response, error) {
					return nil, fmt.Errorf("test delete bucket error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"rb", "cos://test-bucket-1234567890", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("force delete GetBucketType error", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Head",
				func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
					return nil, fmt.Errorf("test head error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"rb", "cos://test-bucket-1234567890", "--force", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("force delete GetBucketVersioning error", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Head",
				func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
					return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
				})
			patches.ApplyMethodFunc(reflect.TypeOf(b), "GetVersioning",
				func(ctx context.Context) (*cos.BucketGetVersionResult, *cos.Response, error) {
					return nil, nil, fmt.Errorf("test get versioning error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"rb", "cos://test-bucket-1234567890", "--force", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("force delete success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Head",
				func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
					return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
				})
			patches.ApplyMethodFunc(reflect.TypeOf(b), "GetVersioning",
				func(ctx context.Context) (*cos.BucketGetVersionResult, *cos.Response, error) {
					return &cos.BucketGetVersionResult{Status: "Suspended"}, &cos.Response{}, nil
				})
			patches.ApplyMethodFunc(reflect.TypeOf(b), "Get",
				func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
					return &cos.BucketGetResult{Contents: []cos.Object{}, IsTruncated: false}, &cos.Response{}, nil
				})
			patches.ApplyMethodFunc(reflect.TypeOf(b), "ListMultipartUploads",
				func(ctx context.Context, opt *cos.ListMultipartUploadsOptions) (*cos.ListMultipartUploadsResult, *cos.Response, error) {
					return &cos.ListMultipartUploadsResult{Uploads: nil, IsTruncated: false}, &cos.Response{}, nil
				})
			patches.ApplyMethodFunc(reflect.TypeOf(b), "Delete",
				func(ctx context.Context, opt ...*cos.BucketDeleteOptions) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"rb", "cos://test-bucket-1234567890", "--force", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("NewClient error", func() {
			patches = ApplyFunc(util.NewClient, func(cfg *util.Config, param *util.Param, bucketName string) (*cos.Client, error) {
				return nil, fmt.Errorf("test NewClient error")
			})
			cmd := rootCmd
			cmd.SetArgs([]string{"rb", "cos://test-bucket-1234567890", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})
	})
}
