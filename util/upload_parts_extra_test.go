package util

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestListParts(t *testing.T) {
	// ListParts 内部调用 CheckUploadExist → GetUploadsListForLs → Bucket.ListMultipartUploads（全局打桩）
	// 以及 GetPartsListForLs → Object.ListParts（全局打桩）
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "test-file.txt"}

	t.Run("uploadId 不存在时返回错误", func(t *testing.T) {
		mockBucketListMultipartUploadsFunc = func(ctx context.Context, opt *cos.ListMultipartUploadsOptions) (*cos.ListMultipartUploadsResult, *cos.Response, error) {
			return &cos.ListMultipartUploadsResult{
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := ListParts(newTestClient(), cosUrl, 10, "nonexistent-upload")
		if err == nil {
			t.Error("期望返回错误（uploadId 不存在），但得到 nil")
		}
		mockBucketListMultipartUploadsFunc = nil
	})

	t.Run("CheckUploadExist 失败时返回错误", func(t *testing.T) {
		mockBucketListMultipartUploadsFunc = func(ctx context.Context, opt *cos.ListMultipartUploadsOptions) (*cos.ListMultipartUploadsResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock list error")
		}
		err := ListParts(newTestClient(), cosUrl, 10, "upload-id")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		mockBucketListMultipartUploadsFunc = nil
	})

	t.Run("ListParts 失败时返回错误", func(t *testing.T) {
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
					{Key: "test-file.txt", UploadID: "upload-id"},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		mockObjectListPartsFunc = func(ctx context.Context, name, uploadID string, opt *cos.ObjectListPartsOptions) (*cos.ObjectListPartsResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock list parts error")
		}
		err := ListParts(newTestClient(), cosUrl, 10, "upload-id")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		mockBucketListMultipartUploadsFunc = nil
		mockObjectListPartsFunc = nil
	})

	t.Run("成功列出分片信息", func(t *testing.T) {
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
					{Key: "test-file.txt", UploadID: "upload-id"},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		mockObjectListPartsFunc = func(ctx context.Context, name, uploadID string, opt *cos.ObjectListPartsOptions) (*cos.ObjectListPartsResult, *cos.Response, error) {
			return &cos.ObjectListPartsResult{
				Parts: []cos.Object{
					{PartNumber: 1, ETag: "etag-001", Size: 5 * 1024 * 1024, LastModified: "2024-01-01T00:00:00Z"},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := ListParts(newTestClient(), cosUrl, 10, "upload-id")
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		mockBucketListMultipartUploadsFunc = nil
		mockObjectListPartsFunc = nil
	})
}
