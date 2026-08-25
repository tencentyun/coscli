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

func TestGetLocalFileKeys(t *testing.T) {
	// getLocalFileKeys 内部调用 GetFileList 和 ReadLocalFileKeys（无 SDK 依赖）

	t.Run("成功读取本地文件 key", func(t *testing.T) {
		tmpDir := filepath.Join(os.TempDir(), "coscli-local-keys-test")
		os.MkdirAll(tmpDir, 0755)
		defer os.RemoveAll(tmpDir)

		// 创建几个本地文件
		for i := 0; i < 3; i++ {
			f, _ := os.CreateTemp(tmpDir, "file-*.txt")
			f.WriteString("content")
			f.Close()
		}

		keys := make(map[string]commonInfoType)
		fileUrl := &FileUrl{urlStr: tmpDir}
		fo := &FileOperations{
			Operation:            Operation{},
			Monitor:              &FileProcessMonitor{},
			SyncDeleteObjectInfo: SyncDeleteObjectInfo{},
		}
		err := getLocalFileKeys(fileUrl, keys, fo, TypeSrc)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		// keys 中应有 3 个文件
		if len(keys) == 0 {
			t.Error("期望 keys 非空")
		}
	})
}

func TestDeleteCosObjectsLargeBatch(t *testing.T) {
	// 测试 DeleteCosObjects 在超过 MaxDeleteBatchCount 时的分批逻辑
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}

	t.Run("批量删除超过阈值时分批", func(t *testing.T) {
		// 创建超过 MaxDeleteBatchCount 的删除项
		keysToDelete := make(map[string]commonInfoType)
		for i := 0; i < MaxDeleteBatchCount+10; i++ {
			key := fmt.Sprintf("file%d.txt", i)
			keysToDelete[key] = commonInfoType{key: key, dir: "prefix/"}
		}
		callCount := 0
		mockDeleteMultiFunc = func(ctx context.Context, opt *cos.ObjectDeleteMultiOptions) (*cos.ObjectDeleteMultiResult, *cos.Response, error) {
			callCount++
			return &cos.ObjectDeleteMultiResult{}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		fo := &FileOperations{
			Operation: Operation{Force: true},
			CpType:    CpTypeCopy,
			Monitor:   &FileProcessMonitor{},
		}
		err := DeleteCosObjects(newTestClient(), keysToDelete, cosUrl, fo)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		// 应该被调用至少 2 次（分批）
		if callCount < 2 {
			t.Errorf("期望 DeleteMulti 被调用 >= 2 次（分批），实际 %d", callCount)
		}
		mockDeleteMultiFunc = nil
	})
}

func TestCheckBackupDirExtra(t *testing.T) {
	t.Run("BackupDir 不为空但不是文件夹时出错", func(t *testing.T) {
		// 创建一个普通文件作为 BackupDir
		tmpFile, _ := os.CreateTemp("", "coscli-backup-file-*.txt")
		tmpFile.Close()
		defer os.Remove(tmpFile.Name())

		tmpDir := filepath.Join(os.TempDir(), "coscli-backup-target-test")
		os.MkdirAll(tmpDir, 0755)
		defer os.RemoveAll(tmpDir)

		fileUrl := &FileUrl{urlStr: tmpDir}
		fo := &FileOperations{
			Operation: Operation{BackupDir: tmpFile.Name()},
		}
		err := CheckBackupDir(fileUrl, fo)
		if err == nil {
			t.Error("期望返回错误（BackupDir 是文件），但得到 nil")
		}
	})
}

func TestGetObjectListByKeys(t *testing.T) {
	t.Run("根据 srcKeys 和 transferKeys 生成 channel", func(t *testing.T) {
		srcKeys := map[string]commonInfoType{
			"file1.txt": {key: "file1.txt", dir: "prefix/", size: 100},
			"file2.txt": {key: "file2.txt", dir: "prefix/", size: 200},
		}
		transferKeys := map[string]commonInfoType{
			"file1.txt": {key: "file1.txt", dir: "prefix/", size: 100},
		}

		chObjects := make(chan objectInfoType, 10)
		chListError := make(chan error, 2)
		fo := &FileOperations{
			Operation: Operation{},
			Monitor:   &FileProcessMonitor{},
		}

		// getObjectListByKeys 会启动 goroutine 调用 freshProgress
		// 需要 chProgressSignal 通道（避免阻塞），也需要启动一个消费者
		chProgressSignal = make(chan chProgressSignalType, 100)
		go func() {
			for range chProgressSignal {
			}
		}()

		go getObjectListByKeys(srcKeys, transferKeys, chObjects, chListError, fo)

		count := 0
		for range chObjects {
			count++
		}
		if count != 2 {
			t.Errorf("期望 2 个对象，实际 %d", count)
		}
	})
}
