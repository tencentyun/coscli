package cmd

import (
	"context"
	"coscli/util"
	"fmt"
	"reflect"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestMbCmd(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("Test coscli mb", t, func() {
		var patches *Patches
		Reset(func() {
			if patches != nil {
				patches.Reset()
				patches = nil
			}
			clearCmd()
		})

		Convey("invalid arguments", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"mb", "cos://test-alias/object", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("invalid tags", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Put",
				func(ctx context.Context, opt *cos.BucketPutOptions) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"mb", "cos://test-bucket-1234567890", "--tags", "invalid tag", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("create bucket success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Put",
				func(ctx context.Context, opt *cos.BucketPutOptions) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"mb", "cos://test-bucket-1234567890", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("create bucket with region success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Put",
				func(ctx context.Context, opt *cos.BucketPutOptions) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"mb", "cos://test-bucket-1234567890", "--region", "ap-guangzhou", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("create ofs bucket success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Put",
				func(ctx context.Context, opt *cos.BucketPutOptions) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"mb", "cos://test-bucket-1234567890", "--ofs", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("create maz bucket success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Put",
				func(ctx context.Context, opt *cos.BucketPutOptions) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"mb", "cos://test-bucket-1234567890", "--maz", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("create bucket error", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "Put",
				func(ctx context.Context, opt *cos.BucketPutOptions) (*cos.Response, error) {
					return nil, fmt.Errorf("test create bucket error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"mb", "cos://test-bucket-1234567890", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("create bucket NewClient error (CreateClient fails)", func() {
			// 直接打桩 util.CreateClient 返回错误
			patches = ApplyFunc(util.CreateClient, func(config *util.Config, param *util.Param, bucketIDName string) (*cos.Client, error) {
				return nil, fmt.Errorf("test CreateClient error")
			})
			cmd := rootCmd
			cmd.SetArgs([]string{"mb", "cos://test-bucket-1234567890", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("no args error", func() {
			// 触发 cobra.ExactArgs(1) 失败分支
			cmd := rootCmd
			cmd.SetArgs([]string{"mb", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})
	})
}
