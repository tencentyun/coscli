package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tencentyun/cos-go-sdk-v5"

	"coscli/util"
)

func TestCatCmd(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("Test coscli cat", t, func() {
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
			cmd.SetArgs([]string{"cat", "invalid-path", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("cat success", func() {
			var o *cos.ObjectService
			patches = ApplyMethodFunc(reflect.TypeOf(o), "Get",
				func(ctx context.Context, name string, opt *cos.ObjectGetOptions, id ...string) (*cos.Response, error) {
					return &cos.Response{
						Response: &http.Response{
							StatusCode: 200,
							Body:       io.NopCloser(strings.NewReader("test content")),
						},
					}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"cat", "cos://test-alias/test.txt", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("cat error", func() {
			var o *cos.ObjectService
			patches = ApplyMethodFunc(reflect.TypeOf(o), "Get",
				func(ctx context.Context, name string, opt *cos.ObjectGetOptions, id ...string) (*cos.Response, error) {
					return nil, fmt.Errorf("test cat error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"cat", "cos://test-alias/obj", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("invalid cos url format", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"cat", "cos:///invalid-object", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("NewClient error", func() {
			patches = ApplyFunc(util.NewClient, func(cfg *util.Config, param *util.Param, bucketName string) (*cos.Client, error) {
				return nil, fmt.Errorf("test NewClient error")
			})
			cmd := rootCmd
			cmd.SetArgs([]string{"cat", "cos://test-alias/obj", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})
	})
}
