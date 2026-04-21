package util

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// mockDeleteMultiFunc 全局 mock 变量，控制 Object.DeleteMulti 行为
var mockDeleteMultiFunc func(ctx context.Context, opt *cos.ObjectDeleteMultiOptions) (*cos.ObjectDeleteMultiResult, *cos.Response, error)

// mockObjectDeleteFunc 全局 mock 变量，控制 Object.Delete 行为
var mockObjectDeleteFunc func(ctx context.Context, name string, opt ...*cos.ObjectDeleteOptions) (*cos.Response, error)

// mockBucketDeleteFunc 全局 mock 变量，控制 Bucket.Delete 行为
var mockBucketDeleteFunc func(ctx context.Context) (*cos.Response, error)

func TestDeleteCosObjects(t *testing.T) {
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}
	fo := &FileOperations{
		Operation: Operation{Force: true}, // Force=true 跳过确认
	}

	t.Run("空 keysToDelete 时直接返回 nil", func(t *testing.T) {
		keysToDelete := map[string]commonInfoType{}
		err := DeleteCosObjects(newTestClient(), keysToDelete, cosUrl, fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("DeleteMulti 调用失败时返回错误", func(t *testing.T) {
		mockDeleteMultiFunc = func(ctx context.Context, opt *cos.ObjectDeleteMultiOptions) (*cos.ObjectDeleteMultiResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock delete multi error")
		}
		defer func() { mockDeleteMultiFunc = nil }()
		keysToDelete := map[string]commonInfoType{
			"file.txt": {key: "file.txt", dir: "prefix/"},
		}
		err := DeleteCosObjects(newTestClient(), keysToDelete, cosUrl, fo)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("DeleteMulti 成功时返回 nil", func(t *testing.T) {
		mockDeleteMultiFunc = func(ctx context.Context, opt *cos.ObjectDeleteMultiOptions) (*cos.ObjectDeleteMultiResult, *cos.Response, error) {
			return &cos.ObjectDeleteMultiResult{}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		defer func() { mockDeleteMultiFunc = nil }()
		keysToDelete := map[string]commonInfoType{
			"file1.txt": {key: "file1.txt", dir: "prefix/"},
			"file2.txt": {key: "file2.txt", dir: "prefix/"},
		}
		err := DeleteCosObjects(newTestClient(), keysToDelete, cosUrl, fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}

func TestDeleteCosObjectVersions(t *testing.T) {
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}
	fo := &FileOperations{
		Operation: Operation{Force: true},
	}

	t.Run("空 keysToDelete 时直接返回 nil", func(t *testing.T) {
		keysToDelete := []cos.Object{}
		err := DeleteCosObjectVersions(newTestClient(), keysToDelete, cosUrl, fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("DeleteMulti 调用失败时返回错误", func(t *testing.T) {
		mockDeleteMultiFunc = func(ctx context.Context, opt *cos.ObjectDeleteMultiOptions) (*cos.ObjectDeleteMultiResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock delete multi error")
		}
		defer func() { mockDeleteMultiFunc = nil }()
		keysToDelete := []cos.Object{
			{Key: "prefix/file.txt", VersionId: "v-001"},
		}
		err := DeleteCosObjectVersions(newTestClient(), keysToDelete, cosUrl, fo)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("DeleteMulti 成功时返回 nil", func(t *testing.T) {
		mockDeleteMultiFunc = func(ctx context.Context, opt *cos.ObjectDeleteMultiOptions) (*cos.ObjectDeleteMultiResult, *cos.Response, error) {
			return &cos.ObjectDeleteMultiResult{}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		defer func() { mockDeleteMultiFunc = nil }()
		keysToDelete := []cos.Object{
			{Key: "prefix/file1.txt", VersionId: "v-001"},
			{Key: "prefix/file2.txt", VersionId: "v-002"},
		}
		err := DeleteCosObjectVersions(newTestClient(), keysToDelete, cosUrl, fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}

func TestRemoveObjectOrVersion(t *testing.T) {
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "file.txt"}

	t.Run("Force=true 且无 versionId 时删除成功", func(t *testing.T) {
		mockObjectDeleteFunc = func(ctx context.Context, name string, opt ...*cos.ObjectDeleteOptions) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 204}}, nil
		}
		defer func() { mockObjectDeleteFunc = nil }()
		fo := &FileOperations{
			Operation: Operation{Force: true, VersionId: ""},
		}
		err := RemoveObjectOrVersion(newTestClient(), cosUrl, fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("Force=true 且有 versionId 时删除成功", func(t *testing.T) {
		mockObjectDeleteFunc = func(ctx context.Context, name string, opt ...*cos.ObjectDeleteOptions) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 204}}, nil
		}
		defer func() { mockObjectDeleteFunc = nil }()
		fo := &FileOperations{
			Operation: Operation{Force: true, VersionId: "v-001"},
		}
		err := RemoveObjectOrVersion(newTestClient(), cosUrl, fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("Object.Delete 调用失败时返回错误", func(t *testing.T) {
		mockObjectDeleteFunc = func(ctx context.Context, name string, opt ...*cos.ObjectDeleteOptions) (*cos.Response, error) {
			return nil, fmt.Errorf("mock delete error")
		}
		defer func() { mockObjectDeleteFunc = nil }()
		fo := &FileOperations{
			Operation: Operation{Force: true},
		}
		err := RemoveObjectOrVersion(newTestClient(), cosUrl, fo)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})
}

func TestCheckBackupDir(t *testing.T) {
	t.Run("目标目录不存在时自动创建", func(t *testing.T) {
		tmpDir := "/tmp/coscli-test-backup-dir-create"
		defer os.RemoveAll(tmpDir)
		fileUrl := &FileUrl{urlStr: tmpDir + "/"}
		fo := &FileOperations{
			Operation: Operation{BackupDir: ""},
		}
		err := CheckBackupDir(fileUrl, fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("目标路径是文件时返回错误", func(t *testing.T) {
		tmpFile := "/tmp/coscli-test-backup-file.txt"
		f, _ := os.Create(tmpFile)
		if f != nil {
			f.Close()
		}
		defer os.Remove(tmpFile)

		fileUrl := &FileUrl{urlStr: tmpFile}
		fo := &FileOperations{
			Operation: Operation{BackupDir: ""},
		}
		err := CheckBackupDir(fileUrl, fo)
		if err == nil {
			t.Error("期望返回错误（路径是文件），但得到 nil")
		}
	})

	t.Run("BackupDir 为空且目录已存在时返回错误", func(t *testing.T) {
		tmpDir := "/tmp/coscli-test-existing-dir"
		_ = os.MkdirAll(tmpDir, 0755)
		defer os.RemoveAll(tmpDir)

		fileUrl := &FileUrl{urlStr: tmpDir + "/"}
		fo := &FileOperations{
			Operation: Operation{BackupDir: ""},
		}
		err := CheckBackupDir(fileUrl, fo)
		if err == nil {
			t.Error("期望返回错误（BackupDir 为空），但得到 nil")
		}
	})

	t.Run("BackupDir 是目标目录的子目录时返回错误", func(t *testing.T) {
		tmpDir := "/tmp/coscli-test-parent-dir"
		backupDir := tmpDir + "/backup/"
		_ = os.MkdirAll(tmpDir, 0755)
		defer os.RemoveAll(tmpDir)

		fileUrl := &FileUrl{urlStr: tmpDir + "/"}
		fo := &FileOperations{
			Operation: Operation{BackupDir: backupDir},
		}
		err := CheckBackupDir(fileUrl, fo)
		if err == nil {
			t.Error("期望返回错误（BackupDir 是子目录），但得到 nil")
		}
	})
}
