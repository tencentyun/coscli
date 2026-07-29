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
)

func TestStatCmd(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("Test coscli stat", t, func() {
		var patches *Patches
		Reset(func() {
			if patches != nil {
				patches.Reset()
				patches = nil
			}
			clearCmd()
		})

		Convey("参数不足", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"stat", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("cos url 格式错误", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"stat", "invalid-path", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("cos url 缺少 object key", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"stat", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("NewClient 失败（未知桶）", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"stat", "cos://unknown-bucket/test.txt", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("SDK Head 调用失败", func() {
			var o *cos.ObjectService
			patches = ApplyMethodFunc(reflect.TypeOf(o), "Head",
				func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
					return nil, fmt.Errorf("mock head error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"stat", "cos://test-alias/test.txt", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("成功查询对象元数据", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Head",
				func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
					return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
				})
			var o *cos.ObjectService
			patches.ApplyMethodFunc(reflect.TypeOf(o), "Head",
				func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
					h := http.Header{}
					h.Set("ETag", `"abc123"`)
					h.Set("Content-Type", "text/plain")
					h.Set("Content-Length", "1024")
					h.Set("Last-Modified", "Wed, 16 Apr 2025 10:00:00 GMT")
					h.Set("x-cos-storage-class", "STANDARD")
					h.Set("x-cos-hash-crc64ecma", "12345678901234")
					h.Set("x-cos-meta-author", "test-user")
					return &cos.Response{Response: &http.Response{StatusCode: 200, Header: h}}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"stat", "cos://test-alias/test.txt", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("带 --version-id 参数成功查询", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Head",
				func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
					return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
				})
			var o *cos.ObjectService
			patches.ApplyMethodFunc(reflect.TypeOf(o), "Head",
				func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
					h := http.Header{}
					h.Set("ETag", `"def456"`)
					h.Set("Content-Type", "application/json")
					h.Set("Content-Length", "512")
					h.Set("Last-Modified", "Wed, 16 Apr 2025 12:00:00 GMT")
					h.Set("x-cos-version-id", "test-version-id-001")
					h.Set("x-cos-storage-class", "STANDARD_IA")
					return &cos.Response{Response: &http.Response{StatusCode: 200, Header: h}}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"stat", "cos://test-alias/test.json", "--version-id", "test-version-id-001", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("带 --version-id 参数 SDK 调用失败", func() {
			var o *cos.ObjectService
			patches = ApplyMethodFunc(reflect.TypeOf(o), "Head",
				func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
					return nil, fmt.Errorf("mock head with version error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"stat", "cos://test-alias/test.txt", "--version-id", "invalid-version", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("cos url 含无效格式（三斜杠）", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"stat", "cos:///invalid-object", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})
	})
}
