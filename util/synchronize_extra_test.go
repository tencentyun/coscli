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

func TestSkipUpload(t *testing.T) {
	// skipUpload 内部调用 GetHead → Object.Head（已全局打桩）
	// 通过 mockHeadFunc 控制行为

	t.Run("SnapshotPath 为空且 Head 返回 404 时不跳过", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 404}}, &cos.ErrorResponse{Response: &http.Response{StatusCode: 404}}
		}
		fo := &FileOperations{
			Operation: Operation{SnapshotPath: ""},
		}
		skip, syncType, err := skipUpload("key", newTestClient(), fo, time.Now().Unix(), "cos-path", "/local/path")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if skip {
			t.Error("期望 skip=false（对象不存在，需要上传）")
		}
		if syncType != SyncTypeCrc64 {
			t.Errorf("期望 syncType=%s，实际 %s", SyncTypeCrc64, syncType)
		}
		mockHeadFunc = nil
	})

	t.Run("Head 调用失败（非 404）时返回错误", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 403}}, fmt.Errorf("access denied")
		}
		fo := &FileOperations{
			Operation: Operation{SnapshotPath: ""},
		}
		_, _, err := skipUpload("key", newTestClient(), fo, time.Now().Unix(), "cos-path", "/local/path")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		mockHeadFunc = nil
	})

	t.Run("IgnoreExisting=true 且对象存在时跳过", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
		}
		fo := &FileOperations{
			Operation: Operation{SnapshotPath: "", IgnoreExisting: true},
		}
		skip, syncType, err := skipUpload("key", newTestClient(), fo, time.Now().Unix(), "cos-path", "/local/path")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if !skip {
			t.Error("期望 skip=true（IgnoreExisting=true）")
		}
		if syncType != SyncTypeIgnoreExisting {
			t.Errorf("期望 syncType=%s，实际 %s", SyncTypeIgnoreExisting, syncType)
		}
		mockHeadFunc = nil
	})
}

func TestSkipDownload(t *testing.T) {
	// skipDownload 内部调用 GetHead → Object.Head（已全局打桩）

	t.Run("时间格式错误时返回错误", func(t *testing.T) {
		fo := &FileOperations{
			Operation: Operation{SnapshotPath: ""},
		}
		_, _, err := skipDownload("key", newTestClient(), fo, "/local/path", "invalid-time-format", "cos-path")
		if err == nil {
			t.Error("期望返回错误（时间格式错误），但得到 nil")
		}
	})

	t.Run("Update=true 且本地文件不存在时返回错误", func(t *testing.T) {
		fo := &FileOperations{
			Operation: Operation{SnapshotPath: "", Update: true},
		}
		modifiedTime := time.Now().UTC().Format(time.RFC3339)
		_, _, err := skipDownload("key", newTestClient(), fo, "/nonexistent/path/file.txt", modifiedTime, "cos-path")
		if err == nil {
			t.Error("期望返回错误（本地文件不存在），但得到 nil")
		}
	})

	t.Run("Update=true 且本地文件更新时跳过", func(t *testing.T) {
		// 创建一个本地文件
		tmpFile, _ := os.CreateTemp("", "coscli-skip-download-*.txt")
		tmpFile.Close()
		defer os.Remove(tmpFile.Name())

		fo := &FileOperations{
			Operation: Operation{SnapshotPath: "", Update: true},
		}
		// 对象修改时间设为过去
		pastTime := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
		skip, syncType, err := skipDownload("key", newTestClient(), fo, tmpFile.Name(), pastTime, "cos-path")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if !skip {
			t.Error("期望 skip=true（本地文件更新）")
		}
		if syncType != SyncTypeUpdate {
			t.Errorf("期望 syncType=%s，实际 %s", SyncTypeUpdate, syncType)
		}
	})
}

func TestSkipCopy(t *testing.T) {
	// skipCopy 内部调用 GetHead → Object.Head（已全局打桩）

	t.Run("目标对象不存在（404）时不跳过", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 404}}, &cos.ErrorResponse{Response: &http.Response{StatusCode: 404}}
		}
		fo := &FileOperations{
			Operation: Operation{},
		}
		skip, err := skipCopy(newTestClient(), newTestClient(), "src-object", "dest-object", fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if skip {
			t.Error("期望 skip=false（目标不存在）")
		}
		mockHeadFunc = nil
	})

	t.Run("Head 调用失败（非 404）时返回错误", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 403}}, fmt.Errorf("access denied")
		}
		fo := &FileOperations{
			Operation: Operation{},
		}
		_, err := skipCopy(newTestClient(), newTestClient(), "src-object", "dest-object", fo)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		mockHeadFunc = nil
	})

	t.Run("IgnoreExisting=true 且目标存在时跳过", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
		}
		fo := &FileOperations{
			Operation: Operation{IgnoreExisting: true},
		}
		skip, err := skipCopy(newTestClient(), newTestClient(), "src-object", "dest-object", fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if !skip {
			t.Error("期望 skip=true（IgnoreExisting=true）")
		}
		mockHeadFunc = nil
	})
}
