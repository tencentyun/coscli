package util

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// mockObjectMultiCopyFunc 已在 copy_test.go 中声明

func TestCosCopySingleObject(t *testing.T) {
	// CosCopy 单对象场景：先调用 Object.Head 获取源对象元信息，再调用 Object.MultiCopy
	srcUrl := &CosUrl{Bucket: "src-bucket", Object: "src-file.txt"}
	destUrl := &CosUrl{Bucket: "dest-bucket", Object: "dest-file.txt"}

	t.Run("Head 返回 404 时返回错误（源对象不存在）", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 404}}, fmt.Errorf("not found")
		}
		fo := &FileOperations{
			CpType:  CpTypeCopy,
			Monitor: &FileProcessMonitor{},
			Operation: Operation{
				Routines: 1,
			},
		}
		err := CosCopy(newTestClient(), newTestClient(), srcUrl, destUrl, fo)
		if err == nil {
			t.Error("期望返回错误（源对象不存在），但得到 nil")
		}
		mockHeadFunc = nil
	})

	t.Run("Head 调用失败（非 404）时返回错误", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 500}}, fmt.Errorf("internal server error")
		}
		fo := &FileOperations{
			CpType:  CpTypeCopy,
			Monitor: &FileProcessMonitor{},
			Operation: Operation{
				Routines: 1,
			},
		}
		err := CosCopy(newTestClient(), newTestClient(), srcUrl, destUrl, fo)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		mockHeadFunc = nil
	})
}

func TestDownloadSingleObject(t *testing.T) {
	// Download 单对象场景：先调用 Object.Head 获取元信息

	t.Run("Head 返回 404 时返回错误", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 404}}, fmt.Errorf("not found")
		}
		cosUrl := &CosUrl{Bucket: "test-bucket", Object: "nonexistent.txt"}
		fileUrl := &FileUrl{urlStr: filepath.Join(os.TempDir(), "download-test.txt")}
		fo := &FileOperations{
			CpType:  CpTypeDownload,
			Monitor: &FileProcessMonitor{},
			Operation: Operation{
				Routines: 1,
			},
		}
		err := Download(newTestClient(), cosUrl, fileUrl, fo)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		mockHeadFunc = nil
	})

	t.Run("Head 调用失败（非 404）时返回错误", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 500}}, fmt.Errorf("internal error")
		}
		cosUrl := &CosUrl{Bucket: "test-bucket", Object: "file.txt"}
		fileUrl := &FileUrl{urlStr: filepath.Join(os.TempDir(), "download-test.txt")}
		fo := &FileOperations{
			CpType:  CpTypeDownload,
			Monitor: &FileProcessMonitor{},
			Operation: Operation{
				Routines: 1,
			},
		}
		err := Download(newTestClient(), cosUrl, fileUrl, fo)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		mockHeadFunc = nil
	})
}

func TestSyncDownload(t *testing.T) {
	// SyncDownload 在 Delete=false 时直接调用 Download

	t.Run("Delete=false 且 Head 返回 404 时返回错误", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 404}}, fmt.Errorf("not found")
		}
		cosUrl := &CosUrl{Bucket: "test-bucket", Object: "nonexistent.txt"}
		fileUrl := &FileUrl{urlStr: filepath.Join(os.TempDir(), "sync-download-test.txt")}
		fo := &FileOperations{
			CpType:  CpTypeDownload,
			Monitor: &FileProcessMonitor{},
			Operation: Operation{
				Routines: 1,
				Delete:   false,
			},
		}
		err := SyncDownload(newTestClient(), cosUrl, fileUrl, fo)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		mockHeadFunc = nil
	})
}

func TestSyncCosCopy(t *testing.T) {
	// SyncCosCopy 在 Delete=false 时直接调用 CosCopy

	t.Run("Delete=false 且 Head 返回 404 时返回错误", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 404}}, fmt.Errorf("not found")
		}
		srcUrl := &CosUrl{Bucket: "src-bucket", Object: "nonexistent.txt"}
		destUrl := &CosUrl{Bucket: "dest-bucket", Object: "dest-file.txt"}
		fo := &FileOperations{
			CpType:  CpTypeCopy,
			Monitor: &FileProcessMonitor{},
			Operation: Operation{
				Routines: 1,
				Delete:   false,
			},
		}
		err := SyncCosCopy(newTestClient(), newTestClient(), srcUrl, destUrl, fo)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		mockHeadFunc = nil
	})
}
