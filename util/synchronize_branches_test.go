package util

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestSkipCopyExtra(t *testing.T) {
	// skipCopy 多分支覆盖

	t.Run("Update=true 且目标对象较新时跳过", func(t *testing.T) {
		callCount := 0
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			callCount++
			headers := http.Header{}
			if callCount == 1 {
				// 目标对象较新
				headers.Set("Last-Modified", time.Now().UTC().Format(time.RFC3339))
			} else {
				// 源对象较旧
				headers.Set("Last-Modified", time.Now().Add(-24*time.Hour).UTC().Format(time.RFC3339))
			}
			return &cos.Response{Response: &http.Response{StatusCode: 200, Header: headers}}, nil
		}
		fo := &FileOperations{
			Operation: Operation{Update: true},
		}
		skip, err := skipCopy(newTestClient(), newTestClient(), "src-object", "dest-object", fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if !skip {
			t.Error("期望 skip=true（目标更新）")
		}
		mockHeadFunc = nil
	})

	t.Run("无选项时 CRC64 相等则跳过", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			headers := http.Header{}
			headers.Set("x-cos-hash-crc64ecma", "12345678")
			return &cos.Response{Response: &http.Response{StatusCode: 200, Header: headers}}, nil
		}
		fo := &FileOperations{
			Operation: Operation{},
		}
		skip, err := skipCopy(newTestClient(), newTestClient(), "src-object", "dest-object", fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if !skip {
			t.Error("期望 skip=true（CRC 相同）")
		}
		mockHeadFunc = nil
	})

	t.Run("无选项时 CRC64 不等则不跳过", func(t *testing.T) {
		callCount := 0
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			callCount++
			headers := http.Header{}
			if callCount == 1 {
				headers.Set("x-cos-hash-crc64ecma", "12345678")
			} else {
				headers.Set("x-cos-hash-crc64ecma", "87654321")
			}
			return &cos.Response{Response: &http.Response{StatusCode: 200, Header: headers}}, nil
		}
		fo := &FileOperations{
			Operation: Operation{},
		}
		skip, err := skipCopy(newTestClient(), newTestClient(), "src-object", "dest-object", fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if skip {
			t.Error("期望 skip=false（CRC 不同）")
		}
		mockHeadFunc = nil
	})
}

func TestSkipUploadExtra(t *testing.T) {
	t.Run("IgnoreExisting=false + Update=true 时本地较旧则跳过", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			headers := http.Header{}
			headers.Set("Last-Modified", time.Now().UTC().Format(time.RFC3339))
			return &cos.Response{Response: &http.Response{StatusCode: 200, Header: headers}}, nil
		}
		fo := &FileOperations{
			Operation: Operation{Update: true},
		}
		// 本地文件比对象旧（本地修改时间更早）
		oldTime := time.Now().Add(-24 * time.Hour).Unix()
		skip, syncType, err := skipUpload("key", newTestClient(), fo, oldTime, "cos-path", "/local/path")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if !skip {
			t.Error("期望 skip=true（本地较旧）")
		}
		if syncType != SyncTypeUpdate {
			t.Errorf("期望 syncType=%s，实际 %s", SyncTypeUpdate, syncType)
		}
		mockHeadFunc = nil
	})
}

func TestSkipDownloadUpdate(t *testing.T) {
	t.Run("Update=true 且本地文件较旧则不跳过", func(t *testing.T) {
		tmpFile, _ := os.CreateTemp("", "coscli-skip-download-*.txt")
		tmpFile.Close()
		defer os.Remove(tmpFile.Name())

		// 将本地文件的修改时间设置为过去
		pastTime := time.Now().Add(-48 * time.Hour)
		os.Chtimes(tmpFile.Name(), pastTime, pastTime)

		fo := &FileOperations{
			Operation: Operation{Update: true},
		}
		// 对象较新
		objectModTime := time.Now().UTC().Format(time.RFC3339)
		skip, _, err := skipDownload("key", newTestClient(), fo, tmpFile.Name(), objectModTime, "cos-path")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if skip {
			t.Error("期望 skip=false（本地较旧，需下载）")
		}
	})
}

func TestTryGetObjectVersionsNon503(t *testing.T) {
	// tryGetObjectVersions 对非 503 错误直接返回

	t.Run("非 503 错误直接返回", func(t *testing.T) {
		mockBucketGetObjectVersionsFunc = func(ctx context.Context, opt *cos.BucketGetObjectVersionsOptions) (*cos.BucketGetObjectVersionsResult, *cos.Response, error) {
			return nil, &cos.Response{Response: &http.Response{StatusCode: 403}}, fmt.Errorf("access denied")
		}
		_, err := tryGetObjectVersions(newTestClient(), &cos.BucketGetObjectVersionsOptions{})
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		mockBucketGetObjectVersionsFunc = nil
	})
}
