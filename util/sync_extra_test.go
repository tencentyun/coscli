package util

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestSyncUploadNoDelete(t *testing.T) {
	// SyncUpload 在 Delete=false 时直接调用 Upload

	t.Run("Delete=false 时正常上传", func(t *testing.T) {
		tmpDir := filepath.Join(os.TempDir(), "coscli-sync-upload-test")
		os.MkdirAll(tmpDir, 0755)
		defer os.RemoveAll(tmpDir)
		f, _ := os.CreateTemp(tmpDir, "file-*.txt")
		f.WriteString("test")
		f.Close()

		mockObjectUploadFunc = func(ctx context.Context, key, localPath string, opt *cos.MultiUploadOptions) (*cos.CompleteMultipartUploadResult, *cos.Response, error) {
			return &cos.CompleteMultipartUploadResult{}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}

		fileUrl := &FileUrl{urlStr: tmpDir + string(os.PathSeparator)}
		cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}
		fo := &FileOperations{
			CpType:  CpTypeUpload,
			Monitor: &FileProcessMonitor{},
			Operation: Operation{
				Routines:    1,
				PartSize:    32,
				Delete:      false,
				ErrRetryNum: 0,
			},
		}
		err := SyncUpload(newTestClient(), fileUrl, cosUrl, fo)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		mockObjectUploadFunc = nil
	})
}

func TestDeleteCosObjectsConfirmFalse(t *testing.T) {
	// DeleteCosObjects 在 Force=false 且 confirm 返回 false 时跳过删除
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}

	t.Run("Force=false 时 confirm 返回 false 跳过删除", func(t *testing.T) {
		called := false
		mockDeleteMultiFunc = func(ctx context.Context, opt *cos.ObjectDeleteMultiOptions) (*cos.ObjectDeleteMultiResult, *cos.Response, error) {
			called = true
			return &cos.ObjectDeleteMultiResult{}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}

		keysToDelete := map[string]commonInfoType{
			"file1.txt": {key: "file1.txt", dir: "prefix/"},
		}
		fo := &FileOperations{
			Operation: Operation{Force: false},
			CpType:    CpTypeCopy,
			Monitor:   &FileProcessMonitor{},
		}
		// confirm 在 Force=false 时需要 stdin 输入，测试环境无输入直接返回 false
		err := DeleteCosObjects(newTestClient(), keysToDelete, cosUrl, fo)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		if called {
			t.Error("期望 DeleteMulti 不被调用（confirm 返回 false）")
		}
		mockDeleteMultiFunc = nil
	})
}

func TestDeleteCosObjectVersionsLargeBatch(t *testing.T) {
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}

	t.Run("版本列表超过阈值时分批", func(t *testing.T) {
		// 构造超过 MaxDeleteBatchCount 的版本列表
		keysToDelete := make([]cos.Object, 0, MaxDeleteBatchCount+5)
		for i := 0; i < MaxDeleteBatchCount+5; i++ {
			keysToDelete = append(keysToDelete, cos.Object{
				Key:       "file.txt",
				VersionId: "v-" + string(rune(i)),
			})
		}
		callCount := 0
		mockDeleteMultiFunc = func(ctx context.Context, opt *cos.ObjectDeleteMultiOptions) (*cos.ObjectDeleteMultiResult, *cos.Response, error) {
			callCount++
			return &cos.ObjectDeleteMultiResult{}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		fo := &FileOperations{
			Operation: Operation{Force: true},
			Monitor:   &FileProcessMonitor{},
		}
		err := DeleteCosObjectVersions(newTestClient(), keysToDelete, cosUrl, fo)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		if callCount < 2 {
			t.Errorf("期望 DeleteMulti 被调用 >= 2 次（分批），实际 %d", callCount)
		}
		mockDeleteMultiFunc = nil
	})
}
