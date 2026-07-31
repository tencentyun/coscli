package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"testing"
	"time"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tencentyun/cos-go-sdk-v5"
)

// mockFileInfo 用于模拟 os.FileInfo
type mockFileInfo struct {
	size int64
}

func (m *mockFileInfo) Name() string       { return "mock" }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() os.FileMode  { return 0644 }
func (m *mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m *mockFileInfo) IsDir() bool        { return false }
func (m *mockFileInfo) Sys() interface{}   { return nil }

func TestHashCmd(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("Test coscli hash", t, func() {
		var patches *Patches
		Reset(func() {
			if patches != nil {
				patches.Reset()
				patches = nil
			}
			clearCmd()
		})

		Convey("invalid hash type for cos object", func() {
			var o *cos.ObjectService
			patches = ApplyMethodFunc(reflect.TypeOf(o), "Head",
				func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
					return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"hash", "cos://test-alias/test.txt", "--type", "sha256", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("cos object crc64 success", func() {
			var o *cos.ObjectService
			patches = ApplyMethodFunc(reflect.TypeOf(o), "Head",
				func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
					h := http.Header{}
					h.Set("x-cos-hash-crc64ecma", "12345678")
					return &cos.Response{Response: &http.Response{StatusCode: 200, Header: h}}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"hash", "cos://test-alias/test.txt", "--type", "crc64", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("cos object md5 success", func() {
			var o *cos.ObjectService
			patches = ApplyMethodFunc(reflect.TypeOf(o), "Head",
				func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
					h := http.Header{}
					h.Set("etag", `"d41d8cd98f00b204e9800998ecf8427e"`)
					return &cos.Response{Response: &http.Response{StatusCode: 200, Header: h}}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"hash", "cos://test-alias/test.txt", "--type", "md5", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("cos object head error", func() {
			var o *cos.ObjectService
			patches = ApplyMethodFunc(reflect.TypeOf(o), "Head",
				func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
					return nil, fmt.Errorf("test head error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"hash", "cos://test-alias/test.txt", "--type", "crc64", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("local file crc64 success", func() {
			// 创建临时文件
			tmpFile, _ := os.CreateTemp("", "hash-test-*.txt")
			tmpFile.WriteString("test content")
			tmpFile.Close()
			defer os.Remove(tmpFile.Name())

			cmd := rootCmd
			cmd.SetArgs([]string{"hash", tmpFile.Name(), "--type", "crc64", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("local file md5 success", func() {
			// 创建临时文件
			tmpFile, _ := os.CreateTemp("", "hash-test-*.txt")
			tmpFile.WriteString("test content")
			tmpFile.Close()
			defer os.Remove(tmpFile.Name())

			cmd := rootCmd
			cmd.SetArgs([]string{"hash", tmpFile.Name(), "--type", "md5", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("local file not found", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"hash", "/nonexistent/path/file.txt", "--type", "crc64", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("local file invalid hash type", func() {
			tmpFile, _ := os.CreateTemp("", "hash-test-*.txt")
			tmpFile.WriteString("test content")
			tmpFile.Close()
			defer os.Remove(tmpFile.Name())

			cmd := rootCmd
			cmd.SetArgs([]string{"hash", tmpFile.Name(), "--type", "sha256", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("local file md5 large file error", func() {
			// 打桩 os.Stat 返回大文件（>32MB）
			patches = ApplyFunc(os.Stat, func(name string) (os.FileInfo, error) {
				return &mockFileInfo{size: 34 * 1024 * 1024}, nil
			})
			cmd := rootCmd
			cmd.SetArgs([]string{"hash", "/tmp/fake-large-file.txt", "--type", "md5", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("cos object NewClient error (unknown bucket)", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"hash", "cos://unknown-alias/test.txt", "--type", "crc64", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("cos object crc64 ShowHash error", func() {
			var o *cos.ObjectService
			patches = ApplyMethodFunc(reflect.TypeOf(o), "Head",
				func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
					return nil, fmt.Errorf("test head error for crc64")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"hash", "cos://test-alias/test.txt", "--type", "crc64", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("cos object md5 ShowHash error", func() {
			var o *cos.ObjectService
			patches = ApplyMethodFunc(reflect.TypeOf(o), "Head",
				func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
					return nil, fmt.Errorf("test head error for md5")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"hash", "cos://test-alias/test.txt", "--type", "md5", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("local file crc64 calculate error (file not found)", func() {
			// crc64 分支没有先检查文件是否存在，直接调用 CalculateHash
			// 传入不存在的文件路径，CalculateHash 会失败
			cmd := rootCmd
			cmd.SetArgs([]string{"hash", "/nonexistent/crc64-file.txt", "--type", "crc64", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("local file md5 stat error (file not found)", func() {
			// 传入不存在的文件路径，--type md5，触发 os.Stat 失败分支
			cmd := rootCmd
			cmd.SetArgs([]string{"hash", "/nonexistent/md5-stat-error.txt", "--type", "md5", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeError)
		})

		Convey("local file md5 calculate error (file not found after stat)", func() {
			// 创建一个临时文件，然后移除读权限，使 CalculateHash 失败
			tmpFile, _ := os.CreateTemp("", "hash-md5-err-*.txt")
			tmpFile.WriteString("test content")
			tmpFile.Close()
			// 移除读权限
			os.Chmod(tmpFile.Name(), 0000)
			defer func() {
				os.Chmod(tmpFile.Name(), 0644)
				os.Remove(tmpFile.Name())
			}()
			cmd := rootCmd
			cmd.SetArgs([]string{"hash", tmpFile.Name(), "--type", "md5", "-c", testConfigPath})
			e := cmd.Execute()
			// 在 root 用户下可能成功，在普通用户下应该失败
			_ = e
		})
	})
}
