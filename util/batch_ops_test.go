package util

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestDeleteKeys(t *testing.T) {
	// deleteKeys 在 Copy/Upload 时调用 DeleteCosObjects，在 Download 时调用 DeleteLocalFiles

	t.Run("CpTypeCopy 时调用 DeleteCosObjects", func(t *testing.T) {
		called := false
		mockDeleteMultiFunc = func(ctx context.Context, opt *cos.ObjectDeleteMultiOptions) (*cos.ObjectDeleteMultiResult, *cos.Response, error) {
			called = true
			return &cos.ObjectDeleteMultiResult{}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		keysToDelete := map[string]commonInfoType{
			"file.txt": {key: "file.txt", dir: ""},
		}
		cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}
		fo := &FileOperations{
			Operation: Operation{Force: true},
			CpType:    CpTypeCopy,
			Monitor:   &FileProcessMonitor{},
		}
		err := deleteKeys(newTestClient(), keysToDelete, cosUrl, fo)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		if !called {
			t.Error("期望 DeleteMulti 被调用")
		}
		mockDeleteMultiFunc = nil
	})

	t.Run("CpTypeDownload 时调用 DeleteLocalFiles", func(t *testing.T) {
		tmpDir := filepath.Join(os.TempDir(), "coscli-deletekeys-test")
		backupDir := filepath.Join(os.TempDir(), "coscli-deletekeys-backup") + string(os.PathSeparator)
		os.MkdirAll(tmpDir, 0755)
		os.MkdirAll(backupDir, 0755)
		defer os.RemoveAll(tmpDir)
		defer os.RemoveAll(backupDir)

		// 创建待删除文件
		srcFile := filepath.Join(tmpDir, "file.txt")
		f, _ := os.Create(srcFile)
		f.Close()

		keysToDelete := map[string]commonInfoType{
			"/file.txt": {key: "/file.txt", dir: ""},
		}
		fileUrl := &FileUrl{urlStr: tmpDir}
		fo := &FileOperations{
			Operation: Operation{BackupDir: backupDir},
			CpType:    CpTypeDownload,
		}
		err := deleteKeys(nil, keysToDelete, fileUrl, fo)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
	})
}

func TestLsAndDuObjects(t *testing.T) {
	// LsAndDuObjects 内部调用 getCosObjectListForLs → Bucket.Get（已全局打桩）
	// 以及 DuObjects → 全局 Bucket.Get 打桩
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: ""}

	t.Run("列出根目录对象和子目录", func(t *testing.T) {
		resetStatisticCounters()
		dirs = nil
		files = nil
		callCount := 0
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			callCount++
			if callCount == 1 {
				// 第一次：根目录
				return &cos.BucketGetResult{
					Contents: []cos.Object{
						{Key: "file1.txt", Size: 100, StorageClass: Standard},
					},
					CommonPrefixes: []string{"subdir/"},
					IsTruncated:    false,
				}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
			}
			// 后续调用：子目录的 DuObjects
			return &cos.BucketGetResult{
				Contents: []cos.Object{
					{Key: "subdir/nested.txt", Size: 50, StorageClass: Standard},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := LsAndDuObjects(newTestClient(), cosUrl, nil, BucketTypeCos)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		mockBucketGetFunc = nil
	})
}

func TestGetOfsObjectList(t *testing.T) {
	// getOfsObjectList 内部调用 tryGetObjects → Bucket.Get（已全局打桩）
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}

	t.Run("扫描 OFS 对象列表", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents: []cos.Object{
					{Key: "prefix/file1.txt", Size: 100, LastModified: "2024-01-01T00:00:00Z"},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		chObjects := make(chan objectInfoType, 10)
		chError := make(chan error, 2)
		fo := &FileOperations{
			Operation: Operation{},
			Monitor:   &FileProcessMonitor{},
		}
		// getOfsObjectList 内部会 close chObjects，所以在 goroutine 中调用
		go getOfsObjectList(newTestClient(), cosUrl, chObjects, chError, fo, false, true)

		// 消费 channel
		count := 0
		for range chObjects {
			count++
		}
		// 等待完成信号
		<-chError
		mockBucketGetFunc = nil
	})
}

func TestConfirmOfs(t *testing.T) {
	// confirmOfs 在 Force=true 时直接返回 true

	t.Run("Force=true 时直接返回 true", func(t *testing.T) {
		fo := &FileOperations{
			Operation: Operation{Force: true},
		}
		cosUrl := &CosUrl{Bucket: "test-bucket"}
		result := confirmOfs("prefix/", fo, cosUrl)
		if !result {
			t.Error("期望 confirmOfs 返回 true")
		}
	})

	t.Run("Force=false 时 stdin 无输入返回 false", func(t *testing.T) {
		fo := &FileOperations{
			Operation: Operation{Force: false},
		}
		cosUrl := &CosUrl{Bucket: "test-bucket"}
		// stdin 无输入时 Scanln 返回 error，confirmOfs 返回 false
		result := confirmOfs("prefix/", fo, cosUrl)
		if result {
			t.Error("期望 confirmOfs 返回 false（无输入）")
		}
	})

	t.Run("Force=false 且 prefix 为空时", func(t *testing.T) {
		fo := &FileOperations{
			Operation: Operation{Force: false},
		}
		cosUrl := &CosUrl{Bucket: "test-bucket"}
		result := confirmOfs("", fo, cosUrl)
		if result {
			t.Error("期望 confirmOfs 返回 false（无输入）")
		}
	})
}
