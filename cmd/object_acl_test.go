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

func TestObjectAclCmd(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("test coscli object_acl", t, func() {
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
			cmd.SetArgs([]string{"object-acl", "--method", "get", "cos:/test-alias/obj", "-c", testConfigPath})
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
			cmd.SetArgs([]string{"object-acl", "--method", "add", "cos://test-alias/obj", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("ofs not support object-acl", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Head",
				func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
					h := http.Header{}
					h.Set("X-Cos-Bucket-Arch", "OFS")
					return &cos.Response{Response: &http.Response{StatusCode: 200, Header: h}}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"object-acl", "--method", "get", "cos://test-alias/obj", "-c", testConfigPath})
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
			patches.ApplyMethodFunc(reflect.TypeOf(o), "PutACL",
				func(ctx context.Context, name string, opt *cos.ObjectPutACLOptions, id ...string) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"object-acl", "--method", "put",
				"cos://test-alias/obj", "--grant-read", `id="100000000003"`, "-c", testConfigPath})
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
			patches.ApplyMethodFunc(reflect.TypeOf(o), "GetACL",
				func(ctx context.Context, name string, id ...string) (*cos.ObjectGetACLResult, *cos.Response, error) {
					return &cos.ObjectGetACLResult{}, &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"object-acl", "--method", "get", "cos://test-alias/obj", "-c", testConfigPath})
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
			cmd.SetArgs([]string{"object-acl", "--method", "get", "cos://test-alias/obj", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("put error", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Head",
				func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
					return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
				})
			var o *cos.ObjectService
			patches.ApplyMethodFunc(reflect.TypeOf(o), "PutACL",
				func(ctx context.Context, name string, opt *cos.ObjectPutACLOptions, id ...string) (*cos.Response, error) {
					return nil, fmt.Errorf("test put acl error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"object-acl", "--method", "put", "cos://test-alias/obj", "--grant-read", `id="100000000003"`, "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("get error", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Head",
				func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
					return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
				})
			var o *cos.ObjectService
			patches.ApplyMethodFunc(reflect.TypeOf(o), "GetACL",
				func(ctx context.Context, name string, id ...string) (*cos.ObjectGetACLResult, *cos.Response, error) {
					return nil, nil, fmt.Errorf("test get acl error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"object-acl", "--method", "get", "cos://test-alias/obj", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("NewClient error", func() {
			patches = ApplyFunc(util.NewClient, func(cfg *util.Config, param *util.Param, bucketName string) (*cos.Client, error) {
				return nil, fmt.Errorf("test NewClient error")
			})
			cmd := rootCmd
			cmd.SetArgs([]string{"object-acl", "--method", "get", "cos://test-alias/obj", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})
	})
}
