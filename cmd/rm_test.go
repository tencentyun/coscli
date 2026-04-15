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

func TestRmCmd(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("Test coscli rm", t, func() {
		var patches *Patches
		Reset(func() {
			if patches != nil {
				patches.Reset()
				patches = nil
			}
			clearCmd()
		})

		Convey("version-id with recursive", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"rm", "cos://test-alias/obj", "--version-id", "v1", "--recursive", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("all-versions without recursive", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"rm", "cos://test-alias/obj", "--all-versions", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("remove single object success", func() {
			// 打桩 ObjectService.Head 返回成功（对象存在）
			var o *cos.ObjectService
			patches = ApplyMethodFunc(reflect.TypeOf(o), "Head",
				func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
					return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
				})
			patches.ApplyMethodFunc(reflect.TypeOf(o), "Delete",
				func(ctx context.Context, name string, opt ...*cos.ObjectDeleteOptions) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"rm", "cos://test-alias/obj", "--force", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("remove single object not found", func() {
			// 打桩 ObjectService.Head 返回 404（对象不存在）
			var o *cos.ObjectService
			patches = ApplyMethodFunc(reflect.TypeOf(o), "Head",
				func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
					resp := &http.Response{StatusCode: 404}
					return nil, &cos.ErrorResponse{Response: resp}
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"rm", "cos://test-alias/obj", "--force", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("remove single object Head error", func() {
			// 打桩 ObjectService.Head 返回非 404 错误
			var o *cos.ObjectService
			patches = ApplyMethodFunc(reflect.TypeOf(o), "Head",
				func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
					return nil, fmt.Errorf("test head error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"rm", "cos://test-alias/obj", "--force", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("remove single object delete error", func() {
			// 打桩 ObjectService.Head 返回成功（对象存在），Delete 返回错误
			// 注意：RemoveObject 忽略了 RemoveObjectOrVersion 的返回值，所以 rm 命令不会返回 Delete 的错误
			var o *cos.ObjectService
			patches = ApplyMethodFunc(reflect.TypeOf(o), "Head",
				func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
					return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
				})
			patches.ApplyMethodFunc(reflect.TypeOf(o), "Delete",
				func(ctx context.Context, name string, opt ...*cos.ObjectDeleteOptions) (*cos.Response, error) {
					return nil, fmt.Errorf("test delete error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"rm", "cos://test-alias/obj", "--force", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("recursive remove success", func() {
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
			cmd.SetArgs([]string{"rm", "cos://test-alias/", "--recursive", "--force", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("no args error", func() {
			// 触发 cobra.MinimumNArgs(1) 失败分支
			cmd := rootCmd
			cmd.SetArgs([]string{"rm", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("invalid bucket name", func() {
			// 传入无效路径，触发 bucketName == "" 分支
			cmd := rootCmd
			cmd.SetArgs([]string{"rm", "invalid-path", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

	})
}
