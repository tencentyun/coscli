package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestSignurlCmd(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("Test coscli signurl", t, func() {
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
			cmd.SetArgs([]string{"signurl", "invalid-path", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("signurl success", func() {
			var o *cos.ObjectService
			patches = ApplyMethodFunc(reflect.TypeOf(o), "GetPresignedURL2",
				func(ctx context.Context, httpMethod string, name string, expired time.Duration, opt interface{}, signHost ...bool) (*url.URL, error) {
					u, _ := url.Parse("https://test-bucket.cos.ap-guangzhou.myqcloud.com/test.txt?sign=xxx")
					return u, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"signurl", "cos://test-alias/test.txt", "--time", "100", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("signurl simple output success", func() {
			var o *cos.ObjectService
			patches = ApplyMethodFunc(reflect.TypeOf(o), "GetPresignedURL2",
				func(ctx context.Context, httpMethod string, name string, expired time.Duration, opt interface{}, signHost ...bool) (*url.URL, error) {
					u, _ := url.Parse("https://test-bucket.cos.ap-guangzhou.myqcloud.com/test.txt?sign=xxx")
					return u, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"signurl", "cos://test-alias/test.txt", "--simple-output", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("signurl error", func() {
			var o *cos.ObjectService
			patches = ApplyMethodFunc(reflect.TypeOf(o), "GetPresignedURL2",
				func(ctx context.Context, httpMethod string, name string, expired time.Duration, opt interface{}, signHost ...bool) (*url.URL, error) {
					return nil, fmt.Errorf("test signurl error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"signurl", "cos://test-alias/test.txt", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("signurl with http header", func() {
			var o *cos.ObjectService
			patches = ApplyMethodFunc(reflect.TypeOf(o), "GetPresignedURL2",
				func(ctx context.Context, httpMethod string, name string, expired time.Duration, opt interface{}, signHost ...bool) (*url.URL, error) {
					u, _ := url.Parse("https://test-bucket.cos.ap-guangzhou.myqcloud.com/test.txt?sign=xxx")
					return u, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"signurl", "cos://test-alias/test.txt",
				"--time", "3600", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("signurl NewClient error (unknown bucket)", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"signurl", "cos://unknown-alias/test.txt", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})
	})
}

// 测试 GetPresignedURL2 的 http.Header 方法打桩
func TestSignurlWithHeaderPatch(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("Test signurl with header patch", t, func() {
		var patches *Patches
		Reset(func() {
			if patches != nil {
				patches.Reset()
				patches = nil
			}
			clearCmd()
		})

		Convey("signurl with header method patch", func() {
			var h http.Header
			patches = ApplyMethodFunc(reflect.TypeOf(h), "Get",
				func(h http.Header, key string) string {
					return ""
				})
			var o *cos.ObjectService
			patches.ApplyMethodFunc(reflect.TypeOf(o), "GetPresignedURL2",
				func(ctx context.Context, httpMethod string, name string, expired time.Duration, opt interface{}, signHost ...bool) (*url.URL, error) {
					u, _ := url.Parse("https://test-bucket.cos.ap-guangzhou.myqcloud.com/test.txt?sign=xxx")
					return u, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"signurl", "cos://test-alias/test.txt", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})
	})
}
