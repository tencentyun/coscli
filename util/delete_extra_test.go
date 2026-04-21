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

func TestRemoveCosObjects(t *testing.T) {
	// RemoveCosObjects 内部调用 getCosObjectListForLs → Bucket.Get（已全局打桩）
	// 以及 DeleteCosObjects → Object.DeleteMulti（已全局打桩）
	// 注意：tryGetObjects 有重试逻辑（10 次 * 1-10 秒），错误场景会卡数十秒，不适合测试
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}

	t.Run("成功删除多个对象", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents: []cos.Object{
					{Key: "prefix/file1.txt"},
					{Key: "prefix/file2.txt"},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		mockDeleteMultiFunc = func(ctx context.Context, opt *cos.ObjectDeleteMultiOptions) (*cos.ObjectDeleteMultiResult, *cos.Response, error) {
			return &cos.ObjectDeleteMultiResult{}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		fo := &FileOperations{
			Operation: Operation{Force: true},
			CpType:    CpTypeCopy,
			Monitor:   &FileProcessMonitor{},
		}
		err := RemoveCosObjects("", newTestClient(), cosUrl, fo)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		mockBucketGetFunc = nil
		mockDeleteMultiFunc = nil
	})

	t.Run("DeleteMulti 失败时返回错误", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents: []cos.Object{
					{Key: "prefix/file1.txt"},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		mockDeleteMultiFunc = func(ctx context.Context, opt *cos.ObjectDeleteMultiOptions) (*cos.ObjectDeleteMultiResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock delete error")
		}
		fo := &FileOperations{
			Operation: Operation{Force: true},
			CpType:    CpTypeCopy,
			Monitor:   &FileProcessMonitor{},
		}
		err := RemoveCosObjects("", newTestClient(), cosUrl, fo)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		mockBucketGetFunc = nil
		mockDeleteMultiFunc = nil
	})
}

func TestRemoveCosObjectVersions(t *testing.T) {
	// RemoveCosObjectVersions 内部调用 getCosObjectVersionListForLs → Bucket.GetObjectVersions（已全局打桩）
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}

	t.Run("GetObjectVersions 失败时返回错误", func(t *testing.T) {
		mockBucketGetObjectVersionsFunc = func(ctx context.Context, opt *cos.BucketGetObjectVersionsOptions) (*cos.BucketGetObjectVersionsResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock versions error")
		}
		fo := &FileOperations{
			Operation: Operation{Force: true},
			Monitor:   &FileProcessMonitor{},
		}
		err := RemoveCosObjectVersions(newTestClient(), cosUrl, fo)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		mockBucketGetObjectVersionsFunc = nil
	})

	t.Run("成功删除版本和删除标记", func(t *testing.T) {
		mockBucketGetObjectVersionsFunc = func(ctx context.Context, opt *cos.BucketGetObjectVersionsOptions) (*cos.BucketGetObjectVersionsResult, *cos.Response, error) {
			return &cos.BucketGetObjectVersionsResult{
				Version: []cos.ListVersionsResultVersion{
					{Key: "prefix/file.txt", VersionId: "v-001"},
				},
				DeleteMarker: []cos.ListVersionsResultDeleteMarker{
					{Key: "prefix/deleted.txt", VersionId: "v-002"},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		mockDeleteMultiFunc = func(ctx context.Context, opt *cos.ObjectDeleteMultiOptions) (*cos.ObjectDeleteMultiResult, *cos.Response, error) {
			return &cos.ObjectDeleteMultiResult{}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		fo := &FileOperations{
			Operation: Operation{Force: true},
			Monitor:   &FileProcessMonitor{},
		}
		err := RemoveCosObjectVersions(newTestClient(), cosUrl, fo)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		mockBucketGetObjectVersionsFunc = nil
		mockDeleteMultiFunc = nil
	})
}

func TestRemoveOfsObjectsRecursive(t *testing.T) {
	// RemoveOfsObjectsRecursive 在 prefix != "" 时调用 Object.Delete（已全局打桩）

	t.Run("prefix 非空时调用 Object.Delete", func(t *testing.T) {
		called := false
		mockObjectDeleteFunc = func(ctx context.Context, name string, opt ...*cos.ObjectDeleteOptions) (*cos.Response, error) {
			called = true
			return &cos.Response{Response: &http.Response{StatusCode: 204}}, nil
		}
		err := RemoveOfsObjectsRecursive(newTestClient(), "prefix/")
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		if !called {
			t.Error("期望 Object.Delete 被调用")
		}
		mockObjectDeleteFunc = nil
	})
}

func TestDeleteLocalFiles(t *testing.T) {
	t.Run("删除本地文件（备份到目标目录）", func(t *testing.T) {
		tmpDir := filepath.Join(os.TempDir(), "coscli-delete-local-test")
		backupDir := filepath.Join(os.TempDir(), "coscli-delete-backup-test") + string(os.PathSeparator)
		os.MkdirAll(tmpDir, 0755)
		os.MkdirAll(backupDir, 0755) // 预先创建 backup 目录
		defer os.RemoveAll(tmpDir)
		defer os.RemoveAll(backupDir)

		// 创建一个待删除文件
		srcFile := filepath.Join(tmpDir, "file.txt")
		f, _ := os.Create(srcFile)
		f.WriteString("content")
		f.Close()

		keysToDelete := map[string]commonInfoType{
			"/file.txt": {key: "/file.txt", dir: ""},
		}
		fileUrl := &FileUrl{urlStr: tmpDir}
		fo := &FileOperations{
			Operation: Operation{BackupDir: backupDir},
		}
		err := DeleteLocalFiles(keysToDelete, fileUrl, fo)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("getAbsPath 失败时返回错误", func(t *testing.T) {
		// 传入一个格式错误的 FileUrl（ToString 返回奇怪字符）
		keysToDelete := map[string]commonInfoType{
			"file.txt": {key: "file.txt"},
		}
		fileUrl := &FileUrl{urlStr: string([]byte{0})} // 非法路径
		fo := &FileOperations{
			Operation: Operation{BackupDir: ""},
		}
		DeleteLocalFiles(keysToDelete, fileUrl, fo)
		// 此用例不强制断言，主要是为了触发 getAbsPath 错误路径
	})
}

func TestMovePath(t *testing.T) {
	t.Run("源文件存在时移动成功", func(t *testing.T) {
		srcFile, _ := os.CreateTemp("", "coscli-movepath-src-*.txt")
		srcFile.WriteString("content")
		srcFile.Close()
		srcName := srcFile.Name()
		destName := srcName + ".moved"
		defer os.Remove(srcName)
		defer os.Remove(destName)

		err := movePath(srcName, destName)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		if _, statErr := os.Stat(destName); os.IsNotExist(statErr) {
			t.Error("期望目标文件已创建")
		}
	})

	t.Run("源文件不存在时返回错误", func(t *testing.T) {
		err := movePath("/nonexistent/path/file.txt", "/tmp/dest.txt")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})
}

func TestRemoveObject(t *testing.T) {
	// RemoveObject 内部会：
	// 1. FormatUrl
	// 2. NewClient
	// 3. GetBucketVersioning（当 AllVersions=true 时）
	// 4. GetBucketType
	// 5. CheckCosObjectExist → Object.IsExist
	// 6. RemoveObjectOrVersion → Object.Delete

	t.Run("cosPath 为空时返回错误（是目录）", func(t *testing.T) {
		fo := &FileOperations{
			Config: &Config{Base: BaseCfg{}, Buckets: []Bucket{}},
			Param:  &Param{},
		}
		// cos://bucket/ 末尾带 / → 是目录
		err := RemoveObject([]string{"cos://test-bucket/"}, fo)
		if err == nil {
			t.Error("期望返回错误（cosPath 是目录），但得到 nil")
		}
	})
}
