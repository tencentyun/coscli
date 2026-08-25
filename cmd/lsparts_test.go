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

func TestLspartsCmd(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("Test coscli lsparts", t, func() {
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
			cmd.SetArgs([]string{"lsparts", "cos://test-alias/", "--limit", "-1", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("cos path error", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"lsparts", "invalid-path", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("list uploads success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "ListMultipartUploads",
				func(ctx context.Context, opt *cos.ListMultipartUploadsOptions) (*cos.ListMultipartUploadsResult, *cos.Response, error) {
					return &cos.ListMultipartUploadsResult{Uploads: nil, IsTruncated: false}, &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"lsparts", "cos://test-alias/", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("list uploads error", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "ListMultipartUploads",
				func(ctx context.Context, opt *cos.ListMultipartUploadsOptions) (*cos.ListMultipartUploadsResult, *cos.Response, error) {
					return nil, nil, fmt.Errorf("test list uploads error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"lsparts", "cos://test-alias/", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("list parts upload not exist", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "ListMultipartUploads",
				func(ctx context.Context, opt *cos.ListMultipartUploadsOptions) (*cos.ListMultipartUploadsResult, *cos.Response, error) {
					return &cos.ListMultipartUploadsResult{Uploads: nil, IsTruncated: false}, &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"lsparts", "cos://test-alias/obj", "--upload-id", "nonexistent-upload-id", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("list parts success", func() {
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
						}{{Key: "obj", UploadID: "test-upload-id"}},
						IsTruncated: false,
					}, &cos.Response{}, nil
				})
			var o *cos.ObjectService
			patches.ApplyMethodFunc(reflect.TypeOf(o), "ListParts",
				func(ctx context.Context, name string, uploadID string, opt *cos.ObjectListPartsOptions) (*cos.ObjectListPartsResult, *cos.Response, error) {
					return &cos.ObjectListPartsResult{Parts: []cos.Object{}, IsTruncated: false}, &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"lsparts", "cos://test-alias/obj", "--upload-id", "test-upload-id", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("invalid cos url format", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"lsparts", "cos:///invalid-object", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("NewClient error", func() {
			patches = ApplyFunc(util.NewClient, func(cfg *util.Config, param *util.Param, bucketName string) (*cos.Client, error) {
				return nil, fmt.Errorf("test NewClient error")
			})
			cmd := rootCmd
			cmd.SetArgs([]string{"lsparts", "cos://test-alias/", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})
	})
}
