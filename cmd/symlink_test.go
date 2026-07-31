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

func TestSymlinkCmd(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("Test coscli symlink", t, func() {
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
			cmd.SetArgs([]string{"symlink", "--method", "create", "invalid-path", "--link", "linkKey", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("invalid method", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"symlink", "--method", "invalid", "cos://test-alias/obj", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("create symlink success", func() {
			var o *cos.ObjectService
			patches = ApplyMethodFunc(reflect.TypeOf(o), "PutSymlink",
				func(ctx context.Context, name string, opt *cos.ObjectPutSymlinkOptions) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"symlink", "--method", "create", "cos://test-alias/obj", "--link", "linkKey", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("create symlink error", func() {
			var o *cos.ObjectService
			patches = ApplyMethodFunc(reflect.TypeOf(o), "PutSymlink",
				func(ctx context.Context, name string, opt *cos.ObjectPutSymlinkOptions) (*cos.Response, error) {
					return nil, fmt.Errorf("test put symlink error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"symlink", "--method", "create", "cos://test-alias/obj", "--link", "linkKey", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("get symlink success", func() {
			var o *cos.ObjectService
			patches = ApplyMethodFunc(reflect.TypeOf(o), "GetSymlink",
				func(ctx context.Context, name string, opt *cos.ObjectGetSymlinkOptions) (string, *cos.Response, error) {
					return "target-object", &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"symlink", "--method", "get", "cos://test-alias/obj", "--link", "linkKey", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("get symlink error", func() {
			var o *cos.ObjectService
			patches = ApplyMethodFunc(reflect.TypeOf(o), "GetSymlink",
				func(ctx context.Context, name string, opt *cos.ObjectGetSymlinkOptions) (string, *cos.Response, error) {
					return "", nil, fmt.Errorf("test get symlink error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"symlink", "--method", "get", "cos://test-alias/obj", "--link", "linkKey", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("invalid cos url format", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"symlink", "--method", "get", "cos:///invalid-object", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("NewClient error", func() {
			patches = ApplyFunc(util.NewClient, func(cfg *util.Config, param *util.Param, bucketName string) (*cos.Client, error) {
				return nil, fmt.Errorf("test NewClient error")
			})
			cmd := rootCmd
			cmd.SetArgs([]string{"symlink", "--method", "get", "cos://test-alias/obj", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})
	})
}
