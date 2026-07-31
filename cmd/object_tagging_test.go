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

func TestObjectTaggingCmd(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("test coscli object_tagging", t, func() {
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
			cmd.SetArgs([]string{"object-tagging", "--method", "get", "cos:/test-alias/obj", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("ofs not support", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Head",
				func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
					h := http.Header{}
					h.Set("X-Cos-Bucket-Arch", "OFS")
					return &cos.Response{Response: &http.Response{StatusCode: 200, Header: h}}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"object-tagging", "--method", "get", "cos://test-alias/obj", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("invalid method", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Head",
				func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
					return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"object-tagging", "--method", "invalid", "cos://test-alias/obj", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("put without tags", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Head",
				func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
					return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"object-tagging", "--method", "put", "cos://test-alias/obj", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("put success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Head",
				func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
					return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
				})
			var o *cos.ObjectService
			patches.ApplyMethodFunc(reflect.TypeOf(o), "PutTagging",
				func(ctx context.Context, name string, opt *cos.ObjectPutTaggingOptions, id ...string) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"object-tagging", "--method", "put", "cos://test-alias/obj", "tag1#test1", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("add success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Head",
				func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
					return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
				})
			var o *cos.ObjectService
			patches.ApplyMethodFunc(reflect.TypeOf(o), "GetTagging",
				func(ctx context.Context, name string, opt ...interface{}) (*cos.ObjectGetTaggingResult, *cos.Response, error) {
					return &cos.ObjectGetTaggingResult{}, &cos.Response{}, nil
				})
			patches.ApplyMethodFunc(reflect.TypeOf(o), "PutTagging",
				func(ctx context.Context, name string, opt *cos.ObjectPutTaggingOptions, id ...string) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"object-tagging", "--method", "add", "cos://test-alias/obj", "tag3#test3", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("get success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Head",
				func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
					return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
				})
			var o *cos.ObjectService
			patches.ApplyMethodFunc(reflect.TypeOf(o), "GetTagging",
				func(ctx context.Context, name string, opt ...interface{}) (*cos.ObjectGetTaggingResult, *cos.Response, error) {
					return &cos.ObjectGetTaggingResult{}, &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"object-tagging", "--method", "get", "cos://test-alias/obj", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("delete all success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Head",
				func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
					return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
				})
			var o *cos.ObjectService
			patches.ApplyMethodFunc(reflect.TypeOf(o), "DeleteTagging",
				func(ctx context.Context, name string, opt ...interface{}) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"object-tagging", "--method", "delete", "cos://test-alias/obj", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("delete specific tags success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Head",
				func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
					return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
				})
			var o *cos.ObjectService
			patches.ApplyMethodFunc(reflect.TypeOf(o), "GetTagging",
				func(ctx context.Context, name string, opt ...interface{}) (*cos.ObjectGetTaggingResult, *cos.Response, error) {
					return &cos.ObjectGetTaggingResult{
						TagSet: []cos.ObjectTaggingTag{{Key: "tag1", Value: "test1"}},
					}, &cos.Response{}, nil
				})
			patches.ApplyMethodFunc(reflect.TypeOf(o), "PutTagging",
				func(ctx context.Context, name string, opt *cos.ObjectPutTaggingOptions, id ...string) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"object-tagging", "--method", "delete", "cos://test-alias/obj", "tag1#test1", "-c", testConfigPath})
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
			cmd.SetArgs([]string{"object-tagging", "--method", "get", "cos://test-alias/obj", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("NewClient error", func() {
			patches = ApplyFunc(util.NewClient, func(cfg *util.Config, param *util.Param, bucketName string) (*cos.Client, error) {
				return nil, fmt.Errorf("test NewClient error")
			})
			cmd := rootCmd
			cmd.SetArgs([]string{"object-tagging", "--method", "get", "cos://test-alias/obj", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})
	})
}
