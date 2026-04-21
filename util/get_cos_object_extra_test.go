package util

import (
	"context"
	"net/http"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestGetCosKeys(t *testing.T) {
	// GetCosKeys 内部调用 getCosObjectList → tryGetObjects → Bucket.Get（已全局打桩）
	// 注意：tryGetObjects 有重试逻辑（10 次 * 1-10 秒），错误场景会卡数十秒，不测试错误场景
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}

	t.Run("成功读取 COS 对象列表", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents: []cos.Object{
					{Key: "prefix/file1.txt", Size: 100, LastModified: "2024-01-01T00:00:00Z"},
					{Key: "prefix/file2.txt", Size: 200, LastModified: "2024-01-02T00:00:00Z"},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		keys := make(map[string]commonInfoType)
		fo := &FileOperations{
			Operation:            Operation{},
			Monitor:              &FileProcessMonitor{},
			SyncDeleteObjectInfo: SyncDeleteObjectInfo{},
		}
		err := GetCosKeys(newTestClient(), cosUrl, keys, fo, TypeSrc)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		mockBucketGetFunc = nil
	})
}

func TestCheckUploadExist(t *testing.T) {
	// CheckUploadExist 内部调用 GetUploadsListForLs → Bucket.ListMultipartUploads（已全局打桩）
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}

	t.Run("uploadId 不存在时返回 false", func(t *testing.T) {
		mockBucketListMultipartUploadsFunc = func(ctx context.Context, opt *cos.ListMultipartUploadsOptions) (*cos.ListMultipartUploadsResult, *cos.Response, error) {
			return &cos.ListMultipartUploadsResult{
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		exist, err := CheckUploadExist(newTestClient(), cosUrl, "nonexistent-upload-id")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if exist {
			t.Error("期望 exist=false")
		}
		mockBucketListMultipartUploadsFunc = nil
	})

	t.Run("uploadId 存在时返回 true", func(t *testing.T) {
		mockBucketListMultipartUploadsFunc = func(ctx context.Context, opt *cos.ListMultipartUploadsOptions) (*cos.ListMultipartUploadsResult, *cos.Response, error) {
			return &cos.ListMultipartUploadsResult{
				Uploads: []struct {
					Key          string
					UploadID     string `xml:"UploadId"`
					StorageClass string
					Initiator    *cos.Initiator
					Owner        *cos.Owner
					Initiated    string
				}{
					{Key: "prefix/file.txt", UploadID: "target-upload-id"},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		exist, err := CheckUploadExist(newTestClient(), cosUrl, "target-upload-id")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if !exist {
			t.Error("期望 exist=true")
		}
		mockBucketListMultipartUploadsFunc = nil
	})
}
