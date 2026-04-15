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

func TestBucketEncryptionCmd(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("test coscli bucket_encryption", t, func() {
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
			cmd.SetArgs([]string{"bucket-encryption", "--method", "get", "cos:/test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("invalid method", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "GetEncryption",
				func(ctx context.Context) (*cos.BucketGetEncryptionResult, *cos.Response, error) {
					return &cos.BucketGetEncryptionResult{}, &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-encryption", "--method", "add", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("put success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "PutEncryption",
				func(ctx context.Context, opt *cos.BucketPutEncryptionOptions) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-encryption", "--method", "put",
				"cos://test-alias", "--sse-algorithm", "AES256", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("get success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "GetEncryption",
				func(ctx context.Context) (*cos.BucketGetEncryptionResult, *cos.Response, error) {
					return &cos.BucketGetEncryptionResult{}, &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-encryption", "--method", "get", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("delete success", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "DeleteEncryption",
				func(ctx context.Context) (*cos.Response, error) {
					return &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-encryption", "--method", "delete", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("put error", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "PutEncryption",
				func(ctx context.Context, opt *cos.BucketPutEncryptionOptions) (*cos.Response, error) {
					return nil, fmt.Errorf("test put encryption error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-encryption", "--method", "put", "cos://test-alias", "--sse-algorithm", "AES256", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("get error", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "GetEncryption",
				func(ctx context.Context) (*cos.BucketGetEncryptionResult, *cos.Response, error) {
					return nil, nil, fmt.Errorf("test get encryption error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-encryption", "--method", "get", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("delete error", func() {
			var b *cos.BucketService
			patches = ApplyMethodFunc(reflect.TypeOf(b), "DeleteEncryption",
				func(ctx context.Context) (*cos.Response, error) {
					return nil, fmt.Errorf("test delete encryption error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-encryption", "--method", "delete", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("NewClient error", func() {
			patches = ApplyFunc(util.NewClient, func(cfg *util.Config, param *util.Param, bucketName string) (*cos.Client, error) {
				return nil, fmt.Errorf("test NewClient error")
			})
			cmd := rootCmd
			cmd.SetArgs([]string{"bucket-encryption", "--method", "get", "cos://test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})
	})
}
