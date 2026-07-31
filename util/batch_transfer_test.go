package util

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestUploadEmptyDir(t *testing.T) {
	// Upload 批量路径：扫描本地空目录，不产生文件

	t.Run("上传空目录时无任何文件操作", func(t *testing.T) {
		tmpDir := filepath.Join(os.TempDir(), "coscli-upload-empty-test")
		os.MkdirAll(tmpDir, 0755)
		defer os.RemoveAll(tmpDir)

		fileUrl := &FileUrl{urlStr: tmpDir + string(os.PathSeparator)}
		cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}
		fo := &FileOperations{
			CpType:  CpTypeUpload,
			Monitor: &FileProcessMonitor{},
			Operation: Operation{
				Routines: 1,
			},
		}
		// Upload 会启动 progressBar goroutine 消费 chProgressSignal
		// 不阻塞，正常完成
		Upload(newTestClient(), fileUrl, cosUrl, fo)
		// 没有任何失败或 panic 即通过
	})
}

func TestUploadSingleFile(t *testing.T) {
	t.Run("上传单个文件（mock Object.Upload）", func(t *testing.T) {
		tmpDir := filepath.Join(os.TempDir(), "coscli-upload-single-test")
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
				ErrRetryNum: 0,
			},
		}
		Upload(newTestClient(), fileUrl, cosUrl, fo)
		mockObjectUploadFunc = nil
	})
}

func TestDownloadEmptyPrefix(t *testing.T) {
	t.Run("下载空前缀时无任何文件下载", func(t *testing.T) {
		tmpDir := filepath.Join(os.TempDir(), "coscli-download-empty-test")
		os.MkdirAll(tmpDir, 0755)
		defer os.RemoveAll(tmpDir)

		// prefix/ 末尾带 / 走批量下载逻辑
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents:    []cos.Object{},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}

		cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}
		fileUrl := &FileUrl{urlStr: tmpDir + string(os.PathSeparator)}
		fo := &FileOperations{
			CpType:  CpTypeDownload,
			Monitor: &FileProcessMonitor{},
			Operation: Operation{
				Routines: 1,
			},
		}
		err := Download(newTestClient(), cosUrl, fileUrl, fo)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		mockBucketGetFunc = nil
	})
}

func TestCosCopyEmptyPrefix(t *testing.T) {
	t.Run("批量 copy 空前缀时无任何对象", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents:    []cos.Object{},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}

		srcUrl := &CosUrl{Bucket: "src-bucket", Object: "prefix/"}
		destUrl := &CosUrl{Bucket: "dest-bucket", Object: "dest-prefix/"}
		fo := &FileOperations{
			CpType:        CpTypeCopy,
			BucketType:    BucketTypeCos,
			Monitor:       &FileProcessMonitor{},
			ErrOutput:     &ErrOutput{Path: "/tmp"},
			ProcessLogger: &ProcessLogger{Path: ""},
			Operation: Operation{
				Routines: 1,
			},
		}
		err := CosCopy(newTestClient(), newTestClient(), srcUrl, destUrl, fo)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		mockBucketGetFunc = nil
	})
}
