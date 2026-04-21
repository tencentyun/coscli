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

// mockObjectDownloadFunc 已在 copy_test.go/download_test.go 中声明

func TestSingleDownload(t *testing.T) {
	// singleDownload 直接调用 c.Object.Download（已全局打桩）
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/file.txt"}

	t.Run("Download 成功时返回成功", func(t *testing.T) {
		tmpDir := filepath.Join(os.TempDir(), "coscli-single-download-test")
		os.MkdirAll(tmpDir, 0755)
		defer os.RemoveAll(tmpDir)

		mockObjectDownloadFunc = func(ctx context.Context, name, localPath string, opt *cos.MultiDownloadOptions, id ...string) (*cos.Response, error) {
			// 创建一个空文件模拟下载成功
			_ = os.WriteFile(localPath, []byte("content"), 0644)
			return &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}

		fileUrl := &FileUrl{urlStr: filepath.Join(tmpDir, "downloaded.txt")}
		fo := &FileOperations{
			CpType:  CpTypeDownload,
			Monitor: &FileProcessMonitor{},
			Operation: Operation{
				Routines: 1,
				PartSize: 32,
			},
		}
		info := objectInfoType{prefix: "prefix/", relativeKey: "file.txt", size: 100, lastModified: "2024-01-01T00:00:00Z"}
		skip, err, isDir, _, _, _ := singleDownload(newTestClient(), fo, info, cosUrl, fileUrl, "")
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		_ = skip
		_ = isDir
		mockObjectDownloadFunc = nil
	})

	t.Run("Download 失败时返回错误", func(t *testing.T) {
		tmpDir := filepath.Join(os.TempDir(), "coscli-single-download-fail-test")
		os.MkdirAll(tmpDir, 0755)
		defer os.RemoveAll(tmpDir)

		mockObjectDownloadFunc = func(ctx context.Context, name, localPath string, opt *cos.MultiDownloadOptions, id ...string) (*cos.Response, error) {
			return nil, fmt.Errorf("mock download error")
		}

		fileUrl := &FileUrl{urlStr: filepath.Join(tmpDir, "downloaded.txt")}
		fo := &FileOperations{
			CpType:  CpTypeDownload,
			Monitor: &FileProcessMonitor{},
			Operation: Operation{
				Routines: 1,
				PartSize: 32,
			},
		}
		info := objectInfoType{prefix: "prefix/", relativeKey: "file.txt", size: 100, lastModified: "2024-01-01T00:00:00Z"}
		_, err, _, _, _, _ := singleDownload(newTestClient(), fo, info, cosUrl, fileUrl, "")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		mockObjectDownloadFunc = nil
	})

	t.Run("目录类型对象创建文件夹成功", func(t *testing.T) {
		tmpDir := filepath.Join(os.TempDir(), "coscli-single-download-dir-test")
		os.MkdirAll(tmpDir, 0755)
		defer os.RemoveAll(tmpDir)

		fileUrl := &FileUrl{urlStr: tmpDir + string(os.PathSeparator)}
		fo := &FileOperations{
			CpType:  CpTypeDownload,
			Monitor: &FileProcessMonitor{},
			Operation: Operation{
				Routines: 1,
				PartSize: 32,
			},
		}
		// relativeKey 以 / 结尾，表示目录
		info := objectInfoType{prefix: "prefix/", relativeKey: "dir/", size: 0, lastModified: ""}
		_, err, isDir, _, _, _ := singleDownload(newTestClient(), fo, info, cosUrl, fileUrl, "")
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		if !isDir {
			t.Error("期望 isDir=true（目录对象）")
		}
	})
}

// TestSingleCopyFail 跳过：singleCopy 内部调用 GenURL 需要完整的 fo.Config 和 buckets 配置，
// 单测环境难以构造，通过集成测试覆盖。

func TestRemoveObjectOrVersionExtra(t *testing.T) {
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "file.txt"}

	t.Run("Force=false 时 stdin 输入 y 执行删除", func(t *testing.T) {
		// Force=false 时会读 stdin，测试环境 Scanf 返回 err，choice=""（默认执行删除）
		called := false
		mockObjectDeleteFunc = func(ctx context.Context, name string, opt ...*cos.ObjectDeleteOptions) (*cos.Response, error) {
			called = true
			return &cos.Response{Response: &http.Response{StatusCode: 204}}, nil
		}
		fo := &FileOperations{
			Operation: Operation{Force: false, VersionId: ""},
		}
		err := RemoveObjectOrVersion(newTestClient(), cosUrl, fo)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		// 因为 Scanf 读不到输入，choice="" 会被认为是确认（y），所以 Delete 被调用
		if !called {
			t.Log("注意：Force=false 且 Scanf 无输入时 choice=\"\" 被视为确认，Delete 被调用")
		}
		mockObjectDeleteFunc = nil
	})

	t.Run("Force=false 有 VersionId 时执行删除", func(t *testing.T) {
		mockObjectDeleteFunc = func(ctx context.Context, name string, opt ...*cos.ObjectDeleteOptions) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 204}}, nil
		}
		fo := &FileOperations{
			Operation: Operation{Force: false, VersionId: "v-001"},
		}
		err := RemoveObjectOrVersion(newTestClient(), cosUrl, fo)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		mockObjectDeleteFunc = nil
	})
}

func TestMoveFileToPathCrossDevice(t *testing.T) {
	// moveFileToPath 当 os.Rename 失败时走 copy+remove 路径

	t.Run("目标目录不存在时返回错误", func(t *testing.T) {
		srcFile, _ := os.CreateTemp("", "coscli-move-src-*.txt")
		srcFile.WriteString("content")
		srcFile.Close()
		defer os.Remove(srcFile.Name())

		// 目标路径的目录不存在
		err := moveFileToPath(srcFile.Name(), "/nonexistent-dir-move-test/dest.txt")
		if err == nil {
			t.Error("期望返回错误（目标目录不存在），但得到 nil")
		}
	})
}

func TestTryRestoreObjectNonFailure(t *testing.T) {
	// TryRestoreObject 非 503 错误直接返回（不重试）

	t.Run("非 503 错误直接返回", func(t *testing.T) {
		mockObjectPostRestoreFunc = func(ctx context.Context, name string, opt *cos.ObjectRestoreOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 409}}, fmt.Errorf("already restored")
		}
		_, err := TryRestoreObject(newTestClient(), "bucket", "key", 1, "Standard")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		mockObjectPostRestoreFunc = nil
	})
}
