package util

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// mockObjectDownloadFunc 全局 mock 变量，控制 Object.Download 行为
var mockObjectDownloadFunc func(ctx context.Context, name, localPath string, opt *cos.MultiDownloadOptions, id ...string) (*cos.Response, error)

func TestDownload(t *testing.T) {
	// Download 内部调用 GetHead（Object.Head 已全局打桩）和 Object.Download（全局打桩）
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "src/file.txt"}
	tmpDir := "/tmp/coscli-download-test"
	_ = os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)
	fileUrl := &FileUrl{urlStr: tmpDir + "/"}

	t.Run("Head 返回 404 时返回 Object not found 错误", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 404}},
				&cos.ErrorResponse{Response: &http.Response{StatusCode: 404}}
		}
		defer func() { mockHeadFunc = nil }()
		fo := &FileOperations{
			CpType:        CpTypeDownload,
			Monitor:       &FileProcessMonitor{},
			ErrOutput:     &ErrOutput{},
			ProcessLogger: &ProcessLogger{},
			Operation:     Operation{Routines: 1},
		}
		err := Download(newTestClient(), cosUrl, fileUrl, fo)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("Head 返回其他错误时返回 Head object err", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return nil, fmt.Errorf("mock head error")
		}
		defer func() { mockHeadFunc = nil }()
		fo := &FileOperations{
			CpType:        CpTypeDownload,
			Monitor:       &FileProcessMonitor{},
			ErrOutput:     &ErrOutput{},
			ProcessLogger: &ProcessLogger{},
			Operation:     Operation{Routines: 1},
		}
		err := Download(newTestClient(), cosUrl, fileUrl, fo)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("Object.Download 成功时返回 nil", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{
				StatusCode:    200,
				Header:        http.Header{"Last-Modified": []string{"Mon, 01 Jan 2023 00:00:00 GMT"}},
				ContentLength: 100,
			}}, nil
		}
		defer func() { mockHeadFunc = nil }()
		mockObjectDownloadFunc = func(ctx context.Context, name, localPath string, opt *cos.MultiDownloadOptions, id ...string) (*cos.Response, error) {
			// 创建一个空文件模拟下载
			f, _ := os.Create(localPath)
			if f != nil {
				f.Close()
			}
			return &cos.Response{Response: &http.Response{
				StatusCode: 200,
				Header:     http.Header{"Last-Modified": []string{"Mon, 01 Jan 2023 00:00:00 GMT"}},
			}}, nil
		}
		defer func() { mockObjectDownloadFunc = nil }()
		fo := &FileOperations{
			CpType:        CpTypeDownload,
			Monitor:       &FileProcessMonitor{},
			ErrOutput:     &ErrOutput{},
			ProcessLogger: &ProcessLogger{},
			Operation:     Operation{Routines: 1, PartSize: 32},
		}
		err := Download(newTestClient(), cosUrl, fileUrl, fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}
