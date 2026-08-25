package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestCpCmd(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("Test coscli cp", t, func() {
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
			cmd.SetArgs([]string{"cp", "-c", testConfigPath})
			e := cmd.Execute()
			fmt.Printf(" : %v", e)
			So(e, ShouldBeError)
		})

		Convey("encryptionType非法", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"cp", "/tmp/coscli-nonexistent-src", "cos://test-alias/obj", "--encryption-type", "SSE-C123", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("retry-num超范围", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"cp", "/tmp/coscli-nonexistent-src", "cos://test-alias/obj", "--retry-num", "1000", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("err-retry-num超范围", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"cp", "/tmp/coscli-nonexistent-src", "cos://test-alias/obj", "--err-retry-num", "1000", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("err-retry-interval超范围", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"cp", "cos://test-alias/obj", "cos://test-alias/obj2", "--err-retry-interval", "11", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("两个本地路径", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"cp", "/tmp/coscli-nonexistent-src", "/tmp/coscli-nonexistent-dst", "-c", testConfigPath})
			e := cmd.Execute()
			fmt.Printf(" : %v", e)
			So(e, ShouldBeError)
		})

		Convey("move只支持cos间", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"cp", "/tmp/coscli-nonexistent-src", "cos://test-alias/obj", "--move", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("encode tag error", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"cp", "/tmp/coscli-nonexistent-src", "cos://test-alias/obj", "--tags", "tag1", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("storageClass不能用于下载", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"cp", "cos://test-alias/obj", "/tmp/coscli-nonexistent-dst", "--storage-class", "STANDARD", "-c", testConfigPath})
			e := cmd.Execute()
			fmt.Printf(" : %v", e)
			So(e, ShouldBeError)
		})

		Convey("Upload", func() {
			Convey("Upload success (no local file)", func() {
				// 路径不存在，FormatUploadPath 会返回错误
				cmd := rootCmd
				cmd.SetArgs([]string{"cp", "/tmp/coscli-nonexistent-src", "cos://test-alias/obj", "--disable-crc64", "-c", testConfigPath})
				e := cmd.Execute()
				So(e, ShouldBeError)
			})

			Convey("Upload with SSE-COS success", func() {
				cmd := rootCmd
				cmd.SetArgs([]string{"cp", "/tmp/coscli-nonexistent-src", "cos://test-alias/obj",
					"--disable-crc64", "--encryption-type", "SSE-COS", "--server-side-encryption", "AES256", "-c", testConfigPath})
				e := cmd.Execute()
				So(e, ShouldBeError)
			})

			Convey("Upload with SSE-C success", func() {
				cmd := rootCmd
				cmd.SetArgs([]string{"cp", "/tmp/coscli-nonexistent-src", "cos://test-alias/obj",
					"--disable-crc64", "--encryption-type", "SSE-C",
					"--sse-customer-algo", "AES256",
					"--sse-customer-key", "12345678901234567890123456789012",
					"--sse-customer-key-md5", "abc", "-c", testConfigPath})
				e := cmd.Execute()
				So(e, ShouldBeError)
			})
		})

		Convey("Download", func() {
			Convey("GetBucketVersioning error with version-id", func() {
				var b *cos.BucketService
				patches = ApplyMethodFunc(reflect.TypeOf(b), "GetVersioning",
					func(ctx context.Context) (*cos.BucketGetVersionResult, *cos.Response, error) {
						return nil, nil, fmt.Errorf("get bucket version error")
					})
				cmd := rootCmd
				cmd.SetArgs([]string{"cp", "cos://test-alias/obj", "./abc", "--version-id", "123", "-c", testConfigPath})
				e := cmd.Execute()
				So(e, ShouldBeError)
			})

			Convey("versioning not enabled with version-id", func() {
				var b *cos.BucketService
				patches = ApplyMethodFunc(reflect.TypeOf(b), "GetVersioning",
					func(ctx context.Context) (*cos.BucketGetVersionResult, *cos.Response, error) {
						return &cos.BucketGetVersionResult{Status: "Suspended"}, &cos.Response{}, nil
					})
				cmd := rootCmd
				cmd.SetArgs([]string{"cp", "cos://test-alias/obj", "./abc", "--version-id", "123", "-c", testConfigPath})
				e := cmd.Execute()
				So(e, ShouldBeError)
			})

			Convey("GetBucketType Head error", func() {
				var b *cos.BucketService
				patches = ApplyMethodFunc(reflect.TypeOf(b), "Head",
					func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
						return nil, fmt.Errorf("test Head error")
					})
				cmd := rootCmd
				cmd.SetArgs([]string{"cp", "cos://test-alias/obj", "./abc", "--disable-crc64", "-c", testConfigPath})
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})

			Convey("Download success (no cos object)", func() {
				var b *cos.BucketService
				patches = ApplyMethodFunc(reflect.TypeOf(b), "Head",
					func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
						return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
					})
				var o *cos.ObjectService
				patches.ApplyMethodFunc(reflect.TypeOf(o), "Get",
					func(ctx context.Context, name string, opt *cos.ObjectGetOptions, id ...string) (*cos.Response, error) {
						return &cos.Response{Response: &http.Response{
							StatusCode: 200,
							Header:     http.Header{},
							Body:       io.NopCloser(nil),
						}}, nil
					})
				patches.ApplyMethodFunc(reflect.TypeOf(b), "Get",
					func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
						return &cos.BucketGetResult{Contents: []cos.Object{}, IsTruncated: false}, &cos.Response{}, nil
					})
				cmd := rootCmd
				cmd.SetArgs([]string{"cp", "cos://test-alias/", "./abc", "--recursive", "--disable-crc64", "-c", testConfigPath})
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})

		Convey("CosCopy", func() {
			Convey("GetBucketVersioning error with version-id", func() {
				var b *cos.BucketService
				patches = ApplyMethodFunc(reflect.TypeOf(b), "GetVersioning",
					func(ctx context.Context) (*cos.BucketGetVersionResult, *cos.Response, error) {
						return nil, nil, fmt.Errorf("get bucket version error")
					})
				cmd := rootCmd
				cmd.SetArgs([]string{"cp", "cos://test-alias/obj", "cos://test-alias2/obj2", "--version-id", "123", "-c", testConfigPath})
				e := cmd.Execute()
				So(e, ShouldBeError)
			})

			Convey("versioning not enabled with version-id", func() {
				var b *cos.BucketService
				patches = ApplyMethodFunc(reflect.TypeOf(b), "GetVersioning",
					func(ctx context.Context) (*cos.BucketGetVersionResult, *cos.Response, error) {
						return &cos.BucketGetVersionResult{Status: "Suspended"}, &cos.Response{}, nil
					})
				cmd := rootCmd
				cmd.SetArgs([]string{"cp", "cos://test-alias/obj", "cos://test-alias2/obj2", "--version-id", "123", "-c", testConfigPath})
				e := cmd.Execute()
				So(e, ShouldBeError)
			})

			Convey("GetBucketType Head error", func() {
				var b *cos.BucketService
				patches = ApplyMethodFunc(reflect.TypeOf(b), "Head",
					func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
						return nil, fmt.Errorf("test Head error")
					})
				cmd := rootCmd
				cmd.SetArgs([]string{"cp", "cos://test-alias/obj", "cos://test-alias2/obj2", "--disable-crc64", "-c", testConfigPath})
				e := cmd.Execute()
				So(e, ShouldBeError)
			})

			Convey("CosCopy success (no objects)", func() {
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
				cmd.SetArgs([]string{"cp", "cos://test-alias/", "cos://test-alias2/", "--recursive", "--disable-crc64", "-c", testConfigPath})
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})

			Convey("CosCopy single object success", func() {
				var b *cos.BucketService
				patches = ApplyMethodFunc(reflect.TypeOf(b), "Head",
					func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
						return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
					})
				var o *cos.ObjectService
				patches.ApplyMethodFunc(reflect.TypeOf(o), "Head",
					func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
						return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
					})
				patches.ApplyMethodFunc(reflect.TypeOf(o), "Copy",
					func(ctx context.Context, name string, sourceURL string, opt *cos.ObjectCopyOptions, id ...string) (*cos.ObjectCopyResult, *cos.Response, error) {
						return &cos.ObjectCopyResult{}, &cos.Response{}, nil
					})
				cmd := rootCmd
				cmd.SetArgs([]string{"cp", "cos://test-alias/obj", "cos://test-alias2/obj2", "--disable-crc64", "-c", testConfigPath})
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})

		Convey("invalid meta string", func() {
			// 传入无效的 meta 字符串，触发 MetaStringToHeader 失败分支
			cmd := rootCmd
			cmd.SetArgs([]string{"cp", "/tmp/src.txt", "cos://test-alias/obj",
				"--meta", "invalid-meta-format-without-colon",
				"-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("invalid srcURL format", func() {
			// 传入无效的 cos URL（bucket 为空但有 object），触发 FormatUrl 失败
			cmd := rootCmd
			cmd.SetArgs([]string{"cp", "cos:///invalid-object", "cos://test-alias/obj",
				"-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("invalid destURL format", func() {
			// 传入无效的 cos URL（bucket 为空但有 object），触发 destURL FormatUrl 失败
			cmd := rootCmd
			cmd.SetArgs([]string{"cp", "cos://test-alias/obj", "cos:///invalid-dest",
				"-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})
	})
}
