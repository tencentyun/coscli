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

func TestBucketInventoryCmd(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("test coscli bucket_inventory", t, func() {
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
			cmd.SetArgs([]string{"inventory", "--method", "list", "cos:/test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("invalid method", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "ListInventoryConfigurations",
				func(ctx context.Context, token string) (*cos.ListBucketInventoryConfigResult, *cos.Response, error) {
					return &cos.ListBucketInventoryConfigResult{IsTruncated: false}, &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"inventory", "--method", "add", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("put success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "PutInventory",
				func(ctx context.Context, id string, opt *cos.BucketPutInventoryOptions) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"inventory", "--method", "put",
				"cos://test-alias", "--task-id", "list4", "--configuration", "<InventoryConfiguration/>", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("get success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "GetInventory",
				func(ctx context.Context, id string) (*cos.BucketGetInventoryResult, *cos.Response, error) {
					return &cos.BucketGetInventoryResult{}, &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"inventory", "--method", "get", "cos://test-alias", "--task-id", "list4", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("list success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "ListInventoryConfigurations",
				func(ctx context.Context, token string) (*cos.ListBucketInventoryConfigResult, *cos.Response, error) {
					return &cos.ListBucketInventoryConfigResult{IsTruncated: false}, &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"inventory", "--method", "list", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("delete success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "DeleteInventory",
				func(ctx context.Context, id string) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"inventory", "--method", "delete", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("post success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "PostInventory",
				func(ctx context.Context, id string, opt *cos.BucketPostInventoryOptions) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"inventory", "--method", "post",
				"cos://test-alias", "--task-id", "list4", "--configuration", "<InventoryConfiguration/>", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("put error", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "PutInventory",
				func(ctx context.Context, id string, opt *cos.BucketPutInventoryOptions) (*cos.Response, error) {
					return nil, fmt.Errorf("test put inventory error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"inventory", "--method", "put", "cos://test-alias", "--task-id", "list4", "--configuration", "<InventoryConfiguration/>", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("get error", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "GetInventory",
				func(ctx context.Context, id string) (*cos.BucketGetInventoryResult, *cos.Response, error) {
					return nil, nil, fmt.Errorf("test get inventory error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"inventory", "--method", "get", "cos://test-alias", "--task-id", "list4", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("list error", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "ListInventoryConfigurations",
				func(ctx context.Context, token string) (*cos.ListBucketInventoryConfigResult, *cos.Response, error) {
					return nil, nil, fmt.Errorf("test list inventory error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"inventory", "--method", "list", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("delete error", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "DeleteInventory",
				func(ctx context.Context, id string) (*cos.Response, error) {
					return nil, fmt.Errorf("test delete inventory error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"inventory", "--method", "delete", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("post error", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "PostInventory",
				func(ctx context.Context, id string, opt *cos.BucketPostInventoryOptions) (*cos.Response, error) {
					return nil, fmt.Errorf("test post inventory error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"inventory", "--method", "post", "cos://test-alias", "--task-id", "list4", "--configuration", "<InventoryConfiguration/>", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("invalid cos url format", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"inventory", "--method", "list", "cos:///invalid-object", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("NewClient error", func() {
			patches = ApplyFunc(util.NewClient, func(cfg *util.Config, param *util.Param, bucketName string) (*cos.Client, error) {
				return nil, fmt.Errorf("test NewClient error")
			})
			cmd := rootCmd
			cmd.SetArgs([]string{"inventory", "--method", "list", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})
	})
}
