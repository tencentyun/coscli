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

// mockObjectAbortMultipartUploadFunc 全局 mock 变量，控制 Object.AbortMultipartUpload 行为
// （Object.AbortMultipartUpload 已在 TestMain 中全局打桩）
var mockObjectAbortMultipartUploadFunc func(ctx context.Context, name, uploadID string) (*cos.Response, error)

func TestGetUploadSnapshotKey(t *testing.T) {
	t.Run("生成上传快照 key", func(t *testing.T) {
		key := getUploadSnapshotKey("/local/path/file.txt", "my-bucket", "prefix/file.txt")
		expected := "/local/path/file.txt" + SnapshotConnector + "cos://my-bucket/prefix/file.txt"
		if key != expected {
			t.Errorf("期望 %q，实际 %q", expected, key)
		}
	})
}

func TestGetDownloadSnapshotKey(t *testing.T) {
	t.Run("生成下载快照 key", func(t *testing.T) {
		key := getDownloadSnapshotKey("/local/path/file.txt", "my-bucket", "prefix/file.txt")
		expected := "cos://my-bucket/prefix/file.txt" + SnapshotConnector + "/local/path/file.txt"
		if key != expected {
			t.Errorf("期望 %q，实际 %q", expected, key)
		}
	})
}

func TestInitSnapshotDb(t *testing.T) {
	t.Run("SnapshotPath 为空时直接返回 nil", func(t *testing.T) {
		fo := &FileOperations{
			Operation: Operation{SnapshotPath: ""},
		}
		err := InitSnapshotDb(nil, nil, fo)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("CpType 为 CpTypeCopy 时返回错误", func(t *testing.T) {
		fo := &FileOperations{
			Operation: Operation{SnapshotPath: "/tmp/snapshot"},
			CpType:    CpTypeCopy,
		}
		err := InitSnapshotDb(nil, nil, fo)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("CpType 为 CpTypeUpload 时打开 leveldb", func(t *testing.T) {
		// snapshotPath 必须与 srcUrl 不是父子关系
		srcDir := filepath.Join(os.TempDir(), "coscli-snapshot-src")
		snapshotDir := filepath.Join(os.TempDir(), "coscli-snapshot-db")
		os.MkdirAll(srcDir, 0755)
		defer os.RemoveAll(srcDir)
		defer os.RemoveAll(snapshotDir)

		srcUrl := &FileUrl{urlStr: srcDir}
		fo := &FileOperations{
			Operation: Operation{SnapshotPath: snapshotDir},
			CpType:    CpTypeUpload,
		}
		err := InitSnapshotDb(srcUrl, nil, fo)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		if fo.SnapshotDb != nil {
			fo.SnapshotDb.Close()
		}
	})
}

func TestRemoveBucket(t *testing.T) {
	// Bucket.Delete 已在 TestMain 中全局打桩，通过 mockBucketDeleteFunc 控制行为

	t.Run("Bucket.Delete 调用失败时返回错误", func(t *testing.T) {
		mockBucketDeleteFunc = func(ctx context.Context) (*cos.Response, error) {
			return nil, fmt.Errorf("mock bucket delete error")
		}
		err := RemoveBucket("test-bucket-123", newTestClient())
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		mockBucketDeleteFunc = nil
	})

	t.Run("Bucket.Delete 成功时返回 nil", func(t *testing.T) {
		mockBucketDeleteFunc = func(ctx context.Context) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 204}}, nil
		}
		err := RemoveBucket("test-bucket-123", newTestClient())
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		mockBucketDeleteFunc = nil
	})
}

func TestGetDirFiles(t *testing.T) {
	t.Run("目录不存在时返回错误", func(t *testing.T) {
		_, err := getDirFiles("/nonexistent/dir", 10)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("空目录 limitCount=0 返回空列表", func(t *testing.T) {
		tmpDir := filepath.Join(os.TempDir(), "coscli-empty-dir-test")
		os.MkdirAll(tmpDir, 0755)
		defer os.RemoveAll(tmpDir)

		// limitCount=0 时 Readdir(0) 返回所有文件，空目录返回空列表无错误
		files, err := getDirFiles(tmpDir, 0)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if len(files) != 0 {
			t.Errorf("期望空列表，实际 %d 个文件", len(files))
		}
	})

	t.Run("有文件的目录返回文件列表", func(t *testing.T) {
		tmpDir := filepath.Join(os.TempDir(), "coscli-dir-files-test")
		os.MkdirAll(tmpDir, 0755)
		defer os.RemoveAll(tmpDir)

		// 创建 3 个文件
		for i := 0; i < 3; i++ {
			f, _ := os.CreateTemp(tmpDir, "file-*.txt")
			f.Close()
		}

		files, err := getDirFiles(tmpDir, 10)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if len(files) != 3 {
			t.Errorf("期望 3 个文件，实际 %d 个", len(files))
		}
	})

	t.Run("limitCount 限制返回数量", func(t *testing.T) {
		tmpDir := filepath.Join(os.TempDir(), "coscli-dir-limit-test")
		os.MkdirAll(tmpDir, 0755)
		defer os.RemoveAll(tmpDir)

		// 创建 5 个文件
		for i := 0; i < 5; i++ {
			f, _ := os.CreateTemp(tmpDir, "file-*.txt")
			f.Close()
		}

		files, err := getDirFiles(tmpDir, 2)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if len(files) != 2 {
			t.Errorf("期望 2 个文件（受 limitCount 限制），实际 %d 个", len(files))
		}
	})
}
