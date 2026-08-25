package util

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

var mockBucketListMultipartUploadsFunc func(ctx context.Context, opt *cos.ListMultipartUploadsOptions) (*cos.ListMultipartUploadsResult, *cos.Response, error)
var mockObjectListPartsFunc func(ctx context.Context, name, uploadID string, opt *cos.ObjectListPartsOptions) (*cos.ObjectListPartsResult, *cos.Response, error)

func TestGetUploadsListForLs(t *testing.T) {
	// Bucket.ListMultipartUploads 已在 TestMain 中全局打桩，通过 mockBucketListMultipartUploadsFunc 控制行为
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}

	t.Run("SDK ListMultipartUploads 调用失败", func(t *testing.T) {
		mockBucketListMultipartUploadsFunc = func(ctx context.Context, opt *cos.ListMultipartUploadsOptions) (*cos.ListMultipartUploadsResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock list uploads error")
		}
		defer func() { mockBucketListMultipartUploadsFunc = nil }()
		err, _, _, _, _ := GetUploadsListForLs(newTestClient(), cosUrl, "", "", 10, true)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK ListMultipartUploads 调用成功（recursive=true）", func(t *testing.T) {
		mockBucketListMultipartUploadsFunc = func(ctx context.Context, opt *cos.ListMultipartUploadsOptions) (*cos.ListMultipartUploadsResult, *cos.Response, error) {
			if opt.Delimiter != "" {
				return nil, nil, fmt.Errorf("期望 delimiter 为空，实际: %s", opt.Delimiter)
			}
			return &cos.ListMultipartUploadsResult{
				Uploads: []struct {
					Key          string
					UploadID     string `xml:"UploadId"`
					StorageClass string
					Initiator    *cos.Initiator
					Owner        *cos.Owner
					Initiated    string
				}{
					{Key: "prefix/file1.txt", UploadID: "upload-id-001"},
					{Key: "prefix/file2.txt", UploadID: "upload-id-002"},
				},
				IsTruncated:        false,
				NextUploadIDMarker: "",
				NextKeyMarker:      "",
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		defer func() { mockBucketListMultipartUploadsFunc = nil }()
		err, uploads, isTruncated, _, _ := GetUploadsListForLs(newTestClient(), cosUrl, "", "", 10, true)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if len(uploads) != 2 {
			t.Errorf("期望 2 个上传任务，实际: %d", len(uploads))
		}
		if isTruncated {
			t.Error("期望 isTruncated=false")
		}
	})

	t.Run("SDK ListMultipartUploads 调用成功（recursive=false，delimiter=/）", func(t *testing.T) {
		mockBucketListMultipartUploadsFunc = func(ctx context.Context, opt *cos.ListMultipartUploadsOptions) (*cos.ListMultipartUploadsResult, *cos.Response, error) {
			if opt.Delimiter != "/" {
				return nil, nil, fmt.Errorf("期望 delimiter=/，实际: %s", opt.Delimiter)
			}
			return &cos.ListMultipartUploadsResult{
				Uploads: []struct {
					Key          string
					UploadID     string `xml:"UploadId"`
					StorageClass string
					Initiator    *cos.Initiator
					Owner        *cos.Owner
					Initiated    string
				}{},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		defer func() { mockBucketListMultipartUploadsFunc = nil }()
		err, _, _, _, _ := GetUploadsListForLs(newTestClient(), cosUrl, "", "", 10, false)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}

func TestGetPartsListForLs(t *testing.T) {
	// Object.ListParts 已在 TestMain 中全局打桩，通过 mockObjectListPartsFunc 控制行为
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "test-file.txt"}

	t.Run("SDK ListParts 调用失败", func(t *testing.T) {
		mockObjectListPartsFunc = func(ctx context.Context, name, uploadID string, opt *cos.ObjectListPartsOptions) (*cos.ObjectListPartsResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock list parts error")
		}
		defer func() { mockObjectListPartsFunc = nil }()
		err, _, _, _ := GetPartsListForLs(newTestClient(), cosUrl, "upload-id-001", "", 10)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK ListParts 调用成功", func(t *testing.T) {
		mockObjectListPartsFunc = func(ctx context.Context, name, uploadID string, opt *cos.ObjectListPartsOptions) (*cos.ObjectListPartsResult, *cos.Response, error) {
			return &cos.ObjectListPartsResult{
				Parts: []cos.Object{
					{PartNumber: 1, ETag: "etag-001", Size: 5 * 1024 * 1024, LastModified: "2023-01-01T00:00:00Z"},
					{PartNumber: 2, ETag: "etag-002", Size: 3 * 1024 * 1024, LastModified: "2023-01-01T00:01:00Z"},
				},
				IsTruncated:          false,
				NextPartNumberMarker: "",
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		defer func() { mockObjectListPartsFunc = nil }()
		err, parts, isTruncated, _ := GetPartsListForLs(newTestClient(), cosUrl, "upload-id-001", "", 10)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if len(parts) != 2 {
			t.Errorf("期望 2 个分片，实际: %d", len(parts))
		}
		if isTruncated {
			t.Error("期望 isTruncated=false")
		}
	})
}

func TestGetUploadsListRecursive(t *testing.T) {
	// Bucket.ListMultipartUploads 已在 TestMain 中全局打桩，通过 mockBucketListMultipartUploadsFunc 控制行为

	t.Run("SDK ListMultipartUploads 调用失败", func(t *testing.T) {
		mockBucketListMultipartUploadsFunc = func(ctx context.Context, opt *cos.ListMultipartUploadsOptions) (*cos.ListMultipartUploadsResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock error")
		}
		defer func() { mockBucketListMultipartUploadsFunc = nil }()
		_, err := GetUploadsListRecursive(newTestClient(), "prefix/", 0, "", "")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("成功获取上传列表（无 include/exclude）", func(t *testing.T) {
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
					{Key: "prefix/file1.txt", UploadID: "upload-id-001", Initiated: "2023-01-01T00:00:00Z"},
					{Key: "prefix/file2.txt", UploadID: "upload-id-002", Initiated: "2023-01-01T00:01:00Z"},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		defer func() { mockBucketListMultipartUploadsFunc = nil }()
		uploads, err := GetUploadsListRecursive(newTestClient(), "prefix/", 0, "", "")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if len(uploads) != 2 {
			t.Errorf("期望 2 个上传任务，实际: %d", len(uploads))
		}
	})

	t.Run("带 include 过滤时只返回匹配的上传任务", func(t *testing.T) {
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
					{Key: "prefix/file1.txt", UploadID: "upload-id-001"},
					{Key: "prefix/file2.log", UploadID: "upload-id-002"},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		defer func() { mockBucketListMultipartUploadsFunc = nil }()
		uploads, err := GetUploadsListRecursive(newTestClient(), "prefix/", 0, ".*\\.txt", "")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if len(uploads) != 1 {
			t.Errorf("期望 1 个上传任务，实际: %d", len(uploads))
		}
	})

	t.Run("带 limit 时只返回指定数量", func(t *testing.T) {
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
					{Key: "prefix/file1.txt", UploadID: "upload-id-001"},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		defer func() { mockBucketListMultipartUploadsFunc = nil }()
		uploads, err := GetUploadsListRecursive(newTestClient(), "prefix/", 5, "", "")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if len(uploads) != 1 {
			t.Errorf("期望 1 个上传任务，实际: %d", len(uploads))
		}
	})
}

func TestListUploads(t *testing.T) {
	// ListUploads 内部调用 GetUploadsListForLs → tryGetUploads → Bucket.ListMultipartUploads（全局打桩）
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}

	t.Run("ListMultipartUploads 调用失败时返回错误", func(t *testing.T) {
		mockBucketListMultipartUploadsFunc = func(ctx context.Context, opt *cos.ListMultipartUploadsOptions) (*cos.ListMultipartUploadsResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock list uploads error")
		}
		defer func() { mockBucketListMultipartUploadsFunc = nil }()
		err := ListUploads(newTestClient(), cosUrl, 10, nil)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("成功列出上传任务（无过滤）", func(t *testing.T) {
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
					{Key: "prefix/file1.txt", UploadID: "upload-id-001", StorageClass: "STANDARD", Initiated: "2023-01-01T00:00:00Z"},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		defer func() { mockBucketListMultipartUploadsFunc = nil }()
		err := ListUploads(newTestClient(), cosUrl, 10, nil)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}

func TestAbortUploads(t *testing.T) {
	// AbortUploads 内部调用 GetUploadsListForLs（全局打桩 Bucket.ListMultipartUploads）
	// 和 Object.AbortMultipartUpload（全局打桩）
	// AbortUploads 内部调用 NewClient，需要有效的 config
	testConfig := &Config{
		Base: BaseCfg{
			SecretID:  "test-secret-id",
			SecretKey: "test-secret-key",
			Protocol:  "https",
		},
		Buckets: []Bucket{
			{
				Name:     "test-bucket-1234567890",
				Alias:    "test-bucket",
				Region:   "ap-guangzhou",
				Endpoint: "cos.ap-guangzhou.myqcloud.com",
			},
		},
	}
	testParam := &Param{}

	t.Run("ListMultipartUploads 调用失败时返回错误", func(t *testing.T) {
		mockBucketListMultipartUploadsFunc = func(ctx context.Context, opt *cos.ListMultipartUploadsOptions) (*cos.ListMultipartUploadsResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock list uploads error")
		}
		defer func() { mockBucketListMultipartUploadsFunc = nil }()
		fo := &FileOperations{
			Config: testConfig,
			Param:  testParam,
		}
		err := AbortUploads([]string{"cos://test-bucket/prefix/"}, fo)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("AbortMultipartUpload 成功时完成清理", func(t *testing.T) {
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
					{Key: "prefix/file1.txt", UploadID: "upload-id-001"},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		defer func() { mockBucketListMultipartUploadsFunc = nil }()
		mockObjectAbortMultipartUploadFunc = func(ctx context.Context, name, uploadID string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 204}}, nil
		}
		defer func() { mockObjectAbortMultipartUploadFunc = nil }()
		fo := &FileOperations{
			Config: testConfig,
			Param:  testParam,
		}
		err := AbortUploads([]string{"cos://test-bucket/prefix/"}, fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("AbortMultipartUpload 失败时记录错误但继续", func(t *testing.T) {
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
					{Key: "prefix/file1.txt", UploadID: "upload-id-001"},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		defer func() { mockBucketListMultipartUploadsFunc = nil }()
		mockObjectAbortMultipartUploadFunc = func(ctx context.Context, name, uploadID string) (*cos.Response, error) {
			return nil, fmt.Errorf("mock abort error")
		}
		defer func() { mockObjectAbortMultipartUploadFunc = nil }()
		fo := &FileOperations{
			Config: testConfig,
			Param:  testParam,
		}
		// AbortUploads 在 abort 失败时只记录日志，不返回错误
		err := AbortUploads([]string{"cos://test-bucket/prefix/"}, fo)
		if err != nil {
			t.Fatalf("期望无错误（abort 失败只记录日志），但得到: %v", err)
		}
	})
}
