package cmd

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestAbortCmd(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("Test coscli abort", t, func() {
		var patches *Patches
		Reset(func() {
			if patches != nil {
				patches.Reset()
				patches = nil
			}
			clearCmd()
		})

		Convey("not enough argument", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"abort", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("0 success 0 fail", func() {
			// 打桩 cos SDK：ListMultipartUploads 返回空列表
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "ListMultipartUploads",
				func(ctx context.Context, opt *cos.ListMultipartUploadsOptions) (*cos.ListMultipartUploadsResult, *cos.Response, error) {
					return &cos.ListMultipartUploadsResult{Uploads: nil, IsTruncated: false}, &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"abort", "cos://test-alias/", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("1 success", func() {
			// 打桩 cos SDK：ListMultipartUploads 返回一个上传任务
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "ListMultipartUploads",
				func(ctx context.Context, opt *cos.ListMultipartUploadsOptions) (*cos.ListMultipartUploadsResult, *cos.Response, error) {
					return &cos.ListMultipartUploadsResult{
						Uploads: []struct {
							Key          string
							UploadID     string `xml:"UploadId"`
							StorageClass string
							Initiator    *cos.Initiator
							Owner        *cos.Owner
							Initiated    string
						}{{Key: "666", UploadID: "888"}},
						IsTruncated: false,
					}, &cos.Response{}, nil
				})
			// 打桩 cos SDK：AbortMultipartUpload 成功
			var o *cos.ObjectService
			patches.ApplyMethodFunc(reflect.TypeOf(o), "AbortMultipartUpload",
				func(ctx context.Context, name string, uploadID string, opt ...*cos.AbortMultipartUploadOptions) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"abort", "cos://test-alias/", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("1 fail", func() {
			// 打桩 cos SDK：ListMultipartUploads 返回一个上传任务
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "ListMultipartUploads",
				func(ctx context.Context, opt *cos.ListMultipartUploadsOptions) (*cos.ListMultipartUploadsResult, *cos.Response, error) {
					return &cos.ListMultipartUploadsResult{
						Uploads: []struct {
							Key          string
							UploadID     string `xml:"UploadId"`
							StorageClass string
							Initiator    *cos.Initiator
							Owner        *cos.Owner
							Initiated    string
						}{{Key: "666", UploadID: "888"}},
						IsTruncated: false,
					}, &cos.Response{}, nil
				})
			// 打桩 cos SDK：AbortMultipartUpload 失败
			var o *cos.ObjectService
			patches.ApplyMethodFunc(reflect.TypeOf(o), "AbortMultipartUpload",
				func(ctx context.Context, name string, uploadID string, opt ...*cos.AbortMultipartUploadOptions) (*cos.Response, error) {
					return nil, fmt.Errorf("test abort fail")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"abort", "cos://test-alias/", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("GetUpload fail", func() {
			// 打桩 cos SDK：ListMultipartUploads 返回错误
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "ListMultipartUploads",
				func(ctx context.Context, opt *cos.ListMultipartUploadsOptions) (*cos.ListMultipartUploadsResult, *cos.Response, error) {
					return nil, nil, fmt.Errorf("test GetUpload client error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"abort", "cos://test-alias/", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})
	})
}
