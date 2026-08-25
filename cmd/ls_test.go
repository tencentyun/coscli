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

func TestLsCmd(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("Test coscli ls", t, func() {
		var patches *Patches
		Reset(func() {
			if patches != nil {
				patches.Reset()
				patches = nil
			}
			clearCmd()
		})

		Convey("invalid limit", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"ls", "--limit", "-2", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("cos path error", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"ls", "invalid-path", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("list buckets success", func() {
			var s *cos.ServiceService
			patches = ApplyMethodFunc(reflect.TypeOf(s), "Get",
				func(ctx context.Context, opt ...*cos.ServiceGetOptions) (*cos.ServiceGetResult, *cos.Response, error) {
					return &cos.ServiceGetResult{Buckets: []cos.Bucket{}, IsTruncated: false}, &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"ls", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("list buckets error", func() {
			var s *cos.ServiceService
			patches = ApplyMethodFunc(reflect.TypeOf(s), "Get",
				func(ctx context.Context, opt ...*cos.ServiceGetOptions) (*cos.ServiceGetResult, *cos.Response, error) {
					return nil, nil, fmt.Errorf("test list buckets error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"ls", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("list objects success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Head",
				func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
					return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
				})
			patches.ApplyMethodFunc(reflect.TypeOf(b), "Get",
				func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
					return &cos.BucketGetResult{Contents: []cos.Object{}, IsTruncated: false}, &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"ls", "cos://test-alias/", "-c", testConfigPath})
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
			cmd.SetArgs([]string{"ls", "cos://test-alias/", "-c", testConfigPath})
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
			cmd.SetArgs([]string{"ls", "cos://test-alias/", "--all-versions", "-c", testConfigPath})
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
			cmd.SetArgs([]string{"ls", "cos://test-alias/", "--all-versions", "-c", testConfigPath})
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
			cmd.SetArgs([]string{"ls", "cos://test-alias/", "--all-versions", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("invalid cos url format error", func() {
			// 传入无效的 cos URL（bucket 为空但有 object），触发 FormatUrl 失败
			cmd := rootCmd
			cmd.SetArgs([]string{"ls", "cos:///invalid-object", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("list buckets NewClient error", func() {
			// 打桩 util.NewClient 返回错误，触发 cosPath == "" 时 NewClient 失败分支
			patches = ApplyFunc(util.NewClient, func(cfg *util.Config, param *util.Param, bucketName string) (*cos.Client, error) {
				return nil, fmt.Errorf("test NewClient error")
			})
			cmd := rootCmd
			cmd.SetArgs([]string{"ls", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("list objects NewClient error", func() {
			// 打桩 util.NewClient 返回错误，触发 cosUrl.IsCosUrl() 时 NewClient 失败分支
			patches = ApplyFunc(util.NewClient, func(cfg *util.Config, param *util.Param, bucketName string) (*cos.Client, error) {
				return nil, fmt.Errorf("test NewClient error")
			})
			cmd := rootCmd
			cmd.SetArgs([]string{"ls", "cos://test-alias/", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})
	})
}
