package util

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

var mockPutSymlinkFunc func(ctx context.Context, name string, opt *cos.ObjectPutSymlinkOptions) (*cos.Response, error)
var mockGetSymlinkFunc func(ctx context.Context, name string, opt *cos.ObjectGetSymlinkOptions) (string, *cos.Response, error)

func TestCreateSymlink(t *testing.T) {
	// Object.PutSymlink 已在 TestMain 中全局打桩，通过 mockPutSymlinkFunc 变量控制行为

	t.Run("SDK PutSymlink 调用失败", func(t *testing.T) {
		mockPutSymlinkFunc = func(ctx context.Context, name string, opt *cos.ObjectPutSymlinkOptions) (*cos.Response, error) {
			return nil, fmt.Errorf("mock put symlink error")
		}
		cosUrl := &CosUrl{Bucket: "test-bucket", Object: "target/file.txt"}
		err := CreateSymlink(newTestClient(), cosUrl, "link/file.txt")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK PutSymlink 调用成功", func(t *testing.T) {
		mockPutSymlinkFunc = func(ctx context.Context, name string, opt *cos.ObjectPutSymlinkOptions) (*cos.Response, error) {
			if name != "link/file.txt" {
				return nil, fmt.Errorf("期望 name=link/file.txt，实际: %s", name)
			}
			if opt.SymlinkTarget != "target/file.txt" {
				return nil, fmt.Errorf("期望 SymlinkTarget=target/file.txt，实际: %s", opt.SymlinkTarget)
			}
			return &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		cosUrl := &CosUrl{Bucket: "test-bucket", Object: "target/file.txt"}
		err := CreateSymlink(newTestClient(), cosUrl, "link/file.txt")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}

func TestGetSymlink(t *testing.T) {
	// Object.GetSymlink 已在 TestMain 中全局打桩，通过 mockGetSymlinkFunc 变量控制行为

	t.Run("SDK GetSymlink 调用失败", func(t *testing.T) {
		mockGetSymlinkFunc = func(ctx context.Context, name string, opt *cos.ObjectGetSymlinkOptions) (string, *cos.Response, error) {
			return "", nil, fmt.Errorf("mock get symlink error")
		}
		res, err := GetSymlink(newTestClient(), "link/file.txt")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		if res != "" {
			t.Errorf("期望 res 为空，实际: %s", res)
		}
	})

	t.Run("SDK GetSymlink 调用成功", func(t *testing.T) {
		mockGetSymlinkFunc = func(ctx context.Context, name string, opt *cos.ObjectGetSymlinkOptions) (string, *cos.Response, error) {
			return "target/file.txt", &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		res, err := GetSymlink(newTestClient(), "link/file.txt")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if res != "target/file.txt" {
			t.Errorf("期望 res=target/file.txt，实际: %s", res)
		}
	})
}
