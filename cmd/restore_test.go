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

func TestRestoreCmd(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("Test coscli restore", t, func() {
		var patches *Patches
		Reset(func() {
			if patches != nil {
				patches.Reset()
				patches = nil
			}
			clearCmd()
		})

		Convey("invalid days", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"restore", "cos://test-alias/obj", "--days", "0", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("cos path error", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"restore", "invalid-path", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("restore single object success", func() {
			var o *cos.ObjectService
			patches = ApplyMethodFunc(reflect.TypeOf(o), "PostRestore",
				func(ctx context.Context, name string, opt *cos.ObjectRestoreOptions, id ...string) (*cos.Response, error) {
					return &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"restore", "cos://test-alias/obj", "--days", "3", "--mode", "Standard", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("restore single object error", func() {
			var o *cos.ObjectService
			patches = ApplyMethodFunc(reflect.TypeOf(o), "PostRestore",
				func(ctx context.Context, name string, opt *cos.ObjectRestoreOptions, id ...string) (*cos.Response, error) {
					return nil, fmt.Errorf("test restore error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"restore", "cos://test-alias/obj", "--days", "3", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("recursive GetBucketType error", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Head",
				func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
					return nil, fmt.Errorf("test head error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"restore", "cos://test-alias/", "--recursive", "--days", "3", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("recursive restore success", func() {
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
			cmd.SetArgs([]string{"restore", "cos://test-alias/", "--recursive", "--days", "3", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("invalid cos url format", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"restore", "cos:///invalid-object", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("NewClient error", func() {
			patches = ApplyFunc(util.NewClient, func(cfg *util.Config, param *util.Param, bucketName string) (*cos.Client, error) {
				return nil, fmt.Errorf("test NewClient error")
			})
			cmd := rootCmd
			cmd.SetArgs([]string{"restore", "cos://test-alias/obj", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})
	})
}
