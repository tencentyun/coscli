package util

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// mockObjectUploadFunc 全局 mock 变量，控制 Object.Upload 行为
// （Object.Upload 已在 TestMain 中全局打桩）
var mockObjectUploadFunc func(ctx context.Context, key, localPath string, opt *cos.MultiUploadOptions) (*cos.CompleteMultipartUploadResult, *cos.Response, error)

// mockObjectPutFunc 全局 mock 变量，控制 Object.Put 行为
// （Object.Put 已在 TestMain 中全局打桩）
var mockObjectPutFunc func(ctx context.Context, name string, r io.Reader, opt *cos.ObjectPutOptions) (*cos.Response, error)

func TestSingleUpload(t *testing.T) {
	// SingleUpload 内部调用 Object.Put（目录）或 Object.Upload（文件），均已全局打桩
	// fileInfoType 字段：filePath（相对路径）、dir（目录前缀）
	// localFilePath = filepath.Join(file.dir, file.filePath)
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}

	t.Run("本地文件不存在时返回错误", func(t *testing.T) {
		fo := &FileOperations{
			CpType:  CpTypeUpload,
			Monitor: &FileProcessMonitor{},
			Operation: Operation{
				Routines: 1,
			},
		}
		// dir="" + filePath="/tmp/coscli-nonexistent-file-12345.txt" → localFilePath="/tmp/coscli-nonexistent-file-12345.txt"
		file := fileInfoType{filePath: "/tmp/coscli-nonexistent-file-12345.txt", dir: ""}
		_, rErr, _, _, _, _ := SingleUpload(newTestClient(), fo, file, cosUrl)
		if rErr == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("上传目录时 isDir=true", func(t *testing.T) {
		tmpDir := "/tmp/coscli-upload-test-dir"
		_ = os.MkdirAll(tmpDir, 0755)
		defer os.RemoveAll(tmpDir)

		mockObjectPutFunc = func(ctx context.Context, name string, r io.Reader, opt *cos.ObjectPutOptions) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}

		fo := &FileOperations{
			CpType:  CpTypeUpload,
			Monitor: &FileProcessMonitor{},
			Operation: Operation{
				Routines: 1,
			},
		}
		// dir="" + filePath=tmpDir → localFilePath=tmpDir（是目录）
		file := fileInfoType{filePath: tmpDir, dir: ""}
		_, rErr, isDir, _, _, _ := SingleUpload(newTestClient(), fo, file, cosUrl)
		if rErr != nil {
			t.Fatalf("期望无错误，但得到: %v", rErr)
		}
		if !isDir {
			t.Error("期望 isDir=true")
		}
		mockObjectPutFunc = nil
	})

	t.Run("上传文件时 Object.Upload 成功", func(t *testing.T) {
		tmpFile := "/tmp/coscli-upload-test-file.txt"
		f, _ := os.Create(tmpFile)
		if f != nil {
			_, _ = f.WriteString("test content")
			f.Close()
		}
		defer os.Remove(tmpFile)

		mockObjectUploadFunc = func(ctx context.Context, key, localPath string, opt *cos.MultiUploadOptions) (*cos.CompleteMultipartUploadResult, *cos.Response, error) {
			return &cos.CompleteMultipartUploadResult{}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}

		fo := &FileOperations{
			CpType:  CpTypeUpload,
			Monitor: &FileProcessMonitor{},
			Operation: Operation{
				Routines: 1,
				PartSize: 32,
			},
		}
		// dir="" + filePath=tmpFile → localFilePath=tmpFile（是文件）
		file := fileInfoType{filePath: tmpFile, dir: ""}
		_, rErr, isDir, _, _, _ := SingleUpload(newTestClient(), fo, file, cosUrl)
		if rErr != nil {
			t.Fatalf("期望无错误，但得到: %v", rErr)
		}
		if isDir {
			t.Error("期望 isDir=false")
		}
		mockObjectUploadFunc = nil
	})

	t.Run("上传文件时 Object.Upload 失败", func(t *testing.T) {
		tmpFile := "/tmp/coscli-upload-test-file2.txt"
		f, _ := os.Create(tmpFile)
		if f != nil {
			_, _ = f.WriteString("test content")
			f.Close()
		}
		defer os.Remove(tmpFile)

		mockObjectUploadFunc = func(ctx context.Context, key, localPath string, opt *cos.MultiUploadOptions) (*cos.CompleteMultipartUploadResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock upload error")
		}

		fo := &FileOperations{
			CpType:  CpTypeUpload,
			Monitor: &FileProcessMonitor{},
			Operation: Operation{
				Routines: 1,
				PartSize: 32,
			},
		}
		file := fileInfoType{filePath: tmpFile, dir: ""}
		_, rErr, _, _, _, _ := SingleUpload(newTestClient(), fo, file, cosUrl)
		if rErr == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		mockObjectUploadFunc = nil
	})

	t.Run("skip=true 时直接跳过", func(t *testing.T) {
		tmpFile := "/tmp/coscli-upload-skip-test.txt"
		f, _ := os.Create(tmpFile)
		if f != nil {
			f.WriteString("skip content")
			f.Close()
		}
		defer os.Remove(tmpFile)

		fo := &FileOperations{
			CpType:  CpTypeUpload,
			Monitor: &FileProcessMonitor{},
			Operation: Operation{
				Routines: 1,
			},
		}
		file := fileInfoType{filePath: tmpFile, dir: "", skip: true}
		skipped, rErr, _, _, _, _ := SingleUpload(newTestClient(), fo, file, cosUrl)
		if rErr != nil {
			t.Fatalf("期望无错误，但得到: %v", rErr)
		}
		if !skipped {
			t.Error("期望 skip=true")
		}
	})
}
