package util

import (
	"context"
	"net/http"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

var mockBucketGetObjectVersionsFunc func(ctx context.Context, opt *cos.BucketGetObjectVersionsOptions) (*cos.BucketGetObjectVersionsResult, *cos.Response, error)

func TestListBuckets(t *testing.T) {
	// Service.Get 已在 TestMain 中全局打桩，通过 mockServiceGetFunc 变量控制行为

	t.Run("SDK Service.Get 调用成功（有桶列表）", func(t *testing.T) {
		mockServiceGetFunc = func(ctx context.Context, opt *cos.ServiceGetOptions) (*cos.ServiceGetResult, *cos.Response, error) {
			return &cos.ServiceGetResult{
				Buckets: []cos.Bucket{
					{Name: "bucket1", Region: "ap-guangzhou", CreationDate: "2024-01-01"},
					{Name: "bucket2", Region: "ap-beijing", CreationDate: "2024-01-02"},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := ListBuckets(newTestClient(), 10)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("limit=0 时不查询", func(t *testing.T) {
		called := false
		mockServiceGetFunc = func(ctx context.Context, opt *cos.ServiceGetOptions) (*cos.ServiceGetResult, *cos.Response, error) {
			called = true
			return &cos.ServiceGetResult{IsTruncated: false}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := ListBuckets(newTestClient(), 0)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		_ = called
	})
}

func TestListObjects(t *testing.T) {
	// Bucket.Get 已在 TestMain 中全局打桩，通过 mockBucketGetFunc 变量控制行为
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}

	t.Run("SDK Bucket.Get 返回空列表", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents:    []cos.Object{},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := ListObjects(newTestClient(), cosUrl, 10, false, nil)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("SDK Bucket.Get 返回对象列表", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents: []cos.Object{
					{Key: "prefix/file1.txt", StorageClass: "STANDARD", LastModified: "2024-01-01T00:00:00Z", ETag: `"abc"`, Size: 1024},
					{Key: "prefix/file2.txt", StorageClass: "STANDARD", LastModified: "2024-01-02T00:00:00Z", ETag: `"def"`, Size: 2048},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := ListObjects(newTestClient(), cosUrl, 10, false, nil)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("SDK Bucket.Get 返回 CommonPrefixes（目录）", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents:       []cos.Object{},
				CommonPrefixes: []string{"prefix/subdir/"},
				IsTruncated:    false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := ListObjects(newTestClient(), cosUrl, 10, false, nil)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("limit=-1 时列出所有对象", func(t *testing.T) {
		callCount := 0
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			callCount++
			if callCount == 1 {
				return &cos.BucketGetResult{
					Contents:    []cos.Object{{Key: "prefix/file1.txt", LastModified: "2024-01-01T00:00:00Z"}},
					IsTruncated: true,
					NextMarker:  "prefix/file1.txt",
				}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
			}
			return &cos.BucketGetResult{
				Contents:    []cos.Object{{Key: "prefix/file2.txt", LastModified: "2024-01-02T00:00:00Z"}},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := ListObjects(newTestClient(), cosUrl, -1, false, nil)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if callCount < 2 {
			t.Errorf("期望至少调用 2 次，实际: %d", callCount)
		}
	})
}

func TestListObjectVersions(t *testing.T) {
	// Bucket.GetObjectVersions 需要打桩
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}

	t.Run("SDK GetObjectVersions 返回版本列表", func(t *testing.T) {
		mockBucketGetObjectVersionsFunc = func(ctx context.Context, opt *cos.BucketGetObjectVersionsOptions) (*cos.BucketGetObjectVersionsResult, *cos.Response, error) {
			return &cos.BucketGetObjectVersionsResult{
				Version: []cos.ListVersionsResultVersion{
					{Key: "prefix/file1.txt", VersionId: "v-001", IsLatest: true, LastModified: "2024-01-01T00:00:00Z", ETag: `"abc"`, Size: 1024},
				},
				DeleteMarker: []cos.ListVersionsResultDeleteMarker{
					{Key: "prefix/file2.txt", VersionId: "v-002", IsLatest: false, LastModified: "2024-01-02T00:00:00Z"},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := ListObjectVersions(newTestClient(), cosUrl, 10, false, nil)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("SDK GetObjectVersions 返回 CommonPrefixes", func(t *testing.T) {
		mockBucketGetObjectVersionsFunc = func(ctx context.Context, opt *cos.BucketGetObjectVersionsOptions) (*cos.BucketGetObjectVersionsResult, *cos.Response, error) {
			return &cos.BucketGetObjectVersionsResult{
				CommonPrefixes: []string{"prefix/subdir/"},
				IsTruncated:    false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := ListObjectVersions(newTestClient(), cosUrl, 10, false, nil)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}
