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

func TestDuCmd(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("Test coscli du", t, func() {
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
			cmd.SetArgs([]string{"du", "invalid-path", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("du success", func() {
			// 打桩 cos SDK：Bucket.Head（GetBucketType内部调用）
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Head",
				func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
					return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
				})
			// 打桩 cos SDK：Bucket.Get（列举对象内部调用）
			patches.ApplyMethodFunc(reflect.TypeOf(b), "Get",
				func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
					return &cos.BucketGetResult{Contents: []cos.Object{}, IsTruncated: false}, &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"du", "cos://test-alias/", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("GetBucketType error", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Head",
				func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
					return nil, fmt.Errorf("test head error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"du", "cos://test-alias/", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("all-versions GetBucketVersioning error", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "GetVersioning",
				func(ctx context.Context) (*cos.BucketGetVersionResult, *cos.Response, error) {
					return nil, nil, fmt.Errorf("test get versioning error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"du", "cos://test-alias/", "--all-versions", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("all-versions versioning not enabled", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "GetVersioning",
				func(ctx context.Context) (*cos.BucketGetVersionResult, *cos.Response, error) {
					return &cos.BucketGetVersionResult{Status: "Suspended"}, &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"du", "cos://test-alias/", "--all-versions", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("all-versions success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "GetVersioning",
				func(ctx context.Context) (*cos.BucketGetVersionResult, *cos.Response, error) {
					return &cos.BucketGetVersionResult{Status: "Enabled"}, &cos.Response{}, nil
				})
			patches.ApplyMethodFunc(reflect.TypeOf(b), "Head",
				func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
					return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
				})
			patches.ApplyMethodFunc(reflect.TypeOf(b), "GetObjectVersions",
				func(ctx context.Context, opt *cos.BucketGetObjectVersionsOptions) (*cos.BucketGetObjectVersionsResult, *cos.Response, error) {
					return &cos.BucketGetObjectVersionsResult{IsTruncated: false}, &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"du", "cos://test-alias/", "--all-versions", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("invalid cos url format", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"du", "cos:///invalid-object", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("NewClient error", func() {
			patches = ApplyFunc(util.NewClient, func(cfg *util.Config, param *util.Param, bucketName string) (*cos.Client, error) {
				return nil, fmt.Errorf("test NewClient error")
			})
			cmd := rootCmd
			cmd.SetArgs([]string{"du", "cos://test-alias/", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})
	})
}
