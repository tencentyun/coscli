package util

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// mockServiceGetFunc 全局 mock 变量，控制 Service.Get 行为
// （Service.Get 已在 TestMain 中全局打桩）
var mockServiceGetFunc func(ctx context.Context, opt *cos.ServiceGetOptions) (*cos.ServiceGetResult, *cos.Response, error)

func TestUrlDecodeCosPattern(t *testing.T) {
	t.Run("URL 解码对象 Key", func(t *testing.T) {
		objects := []cos.Object{
			{Key: "path%2Fto%2Ffile.txt"},
			{Key: "normal-key.txt"},
			{Key: "%E4%B8%AD%E6%96%87.txt"},
		}
		result := UrlDecodeCosPattern(objects)
		if len(result) != 3 {
			t.Fatalf("期望 3 个对象，实际 %d", len(result))
		}
		if result[0].Key != "path/to/file.txt" {
			t.Errorf("期望 path/to/file.txt，实际 %s", result[0].Key)
		}
		if result[1].Key != "normal-key.txt" {
			t.Errorf("期望 normal-key.txt，实际 %s", result[1].Key)
		}
		if result[2].Key != "中文.txt" {
			t.Errorf("期望 中文.txt，实际 %s", result[2].Key)
		}
	})

	t.Run("空列表返回空列表", func(t *testing.T) {
		result := UrlDecodeCosPattern([]cos.Object{})
		if len(result) != 0 {
			t.Errorf("期望空列表，实际 %d 个", len(result))
		}
	})
}

func TestMatchCosPattern(t *testing.T) {
	objects := []cos.Object{
		{Key: "images/photo.jpg"},
		{Key: "docs/readme.txt"},
		{Key: "images/icon.png"},
		{Key: "data/file.csv"},
	}

	t.Run("include=true 匹配 jpg 文件", func(t *testing.T) {
		result := MatchCosPattern(objects, `\.jpg$`, true)
		if len(result) != 1 {
			t.Fatalf("期望 1 个匹配，实际 %d", len(result))
		}
		if result[0].Key != "images/photo.jpg" {
			t.Errorf("期望 images/photo.jpg，实际 %s", result[0].Key)
		}
	})

	t.Run("include=false 排除 images 目录", func(t *testing.T) {
		result := MatchCosPattern(objects, `^images/`, false)
		if len(result) != 2 {
			t.Fatalf("期望 2 个结果，实际 %d", len(result))
		}
	})

	t.Run("无匹配时返回空列表", func(t *testing.T) {
		result := MatchCosPattern(objects, `\.mp4$`, true)
		if len(result) != 0 {
			t.Errorf("期望 0 个匹配，实际 %d", len(result))
		}
	})
}

func TestMatchUploadPattern(t *testing.T) {
	uploads := []UploadInfo{
		{Key: "video/movie.mp4"},
		{Key: "docs/report.pdf"},
		{Key: "video/clip.mp4"},
	}

	t.Run("include=true 匹配 mp4 文件", func(t *testing.T) {
		result := MatchUploadPattern(uploads, `\.mp4$`, true)
		if len(result) != 2 {
			t.Fatalf("期望 2 个匹配，实际 %d", len(result))
		}
	})

	t.Run("include=false 排除 video 目录", func(t *testing.T) {
		result := MatchUploadPattern(uploads, `^video/`, false)
		if len(result) != 1 {
			t.Fatalf("期望 1 个结果，实际 %d", len(result))
		}
		if result[0].Key != "docs/report.pdf" {
			t.Errorf("期望 docs/report.pdf，实际 %s", result[0].Key)
		}
	})
}

func TestTryGetObjectVersions(t *testing.T) {
	// Bucket.GetObjectVersions 已在 TestMain 中全局打桩，通过 mockBucketGetObjectVersionsFunc 控制行为

	t.Run("非 503 错误直接返回", func(t *testing.T) {
		mockBucketGetObjectVersionsFunc = func(ctx context.Context, opt *cos.BucketGetObjectVersionsOptions) (*cos.BucketGetObjectVersionsResult, *cos.Response, error) {
			return nil, &cos.Response{Response: &http.Response{StatusCode: 403}}, fmt.Errorf("access denied")
		}
		opt := &cos.BucketGetObjectVersionsOptions{Prefix: "test/"}
		_, err := tryGetObjectVersions(newTestClient(), opt)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		mockBucketGetObjectVersionsFunc = nil
	})

	t.Run("成功返回版本列表", func(t *testing.T) {
		mockBucketGetObjectVersionsFunc = func(ctx context.Context, opt *cos.BucketGetObjectVersionsOptions) (*cos.BucketGetObjectVersionsResult, *cos.Response, error) {
			return &cos.BucketGetObjectVersionsResult{
				Version: []cos.ListVersionsResultVersion{
					{Key: "file.txt", VersionId: "v-001"},
				},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		opt := &cos.BucketGetObjectVersionsOptions{Prefix: "test/"}
		res, err := tryGetObjectVersions(newTestClient(), opt)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if len(res.Version) != 1 {
			t.Errorf("期望 1 个版本，实际 %d", len(res.Version))
		}
		mockBucketGetObjectVersionsFunc = nil
	})
}

func TestTryGetObjects(t *testing.T) {
	// Bucket.Get 已在 TestMain 中全局打桩，通过 mockBucketGetFunc 控制行为

	t.Run("5xx 错误不叠加应用层重试，仅调用一次即返回错误", func(t *testing.T) {
		// 复现 bug：list 阶段遇到 5xx 时，SDK 层已重试过，
		// 应用层 tryGetObjects 不应再叠加重试，否则会长时间/无限重试导致进程 hang。
		callCount := 0
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			callCount++
			return nil, &cos.Response{Response: &http.Response{StatusCode: 500}},
				&cos.ErrorResponse{Response: &http.Response{StatusCode: 500}, Code: "InternalError"}
		}
		opt := &cos.BucketGetOptions{Prefix: "test/"}
		_, err := tryGetObjects(newTestClient(), opt)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		if callCount != 1 {
			t.Errorf("期望 5xx 时只调用一次 Bucket.Get（不叠加重试），实际调用 %d 次", callCount)
		}
		mockBucketGetFunc = nil
	})

	t.Run("503 错误同样不叠加应用层重试", func(t *testing.T) {
		callCount := 0
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			callCount++
			return nil, &cos.Response{Response: &http.Response{StatusCode: 503}},
				&cos.ErrorResponse{Response: &http.Response{StatusCode: 503}}
		}
		opt := &cos.BucketGetOptions{Prefix: "test/"}
		_, err := tryGetObjects(newTestClient(), opt)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		if callCount != 1 {
			t.Errorf("期望 503 时只调用一次 Bucket.Get，实际调用 %d 次", callCount)
		}
		mockBucketGetFunc = nil
	})

	t.Run("成功返回对象列表", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents: []cos.Object{
					{Key: "test/file.txt"},
				},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		opt := &cos.BucketGetOptions{Prefix: "test/"}
		res, err := tryGetObjects(newTestClient(), opt)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if len(res.Contents) != 1 {
			t.Errorf("期望 1 个对象，实际 %d", len(res.Contents))
		}
		mockBucketGetFunc = nil
	})
}

func TestGetBucketsList(t *testing.T) {
	// Service.Get 已在 TestMain 中全局打桩，通过 mockServiceGetFunc 控制行为

	t.Run("Service.Get 调用失败时返回错误", func(t *testing.T) {
		mockServiceGetFunc = func(ctx context.Context, opt *cos.ServiceGetOptions) (*cos.ServiceGetResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock service get error")
		}
		_, _, _, err := GetBucketsList(newTestClient(), 10, "")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		mockServiceGetFunc = nil
	})

	t.Run("成功返回桶列表", func(t *testing.T) {
		mockServiceGetFunc = func(ctx context.Context, opt *cos.ServiceGetOptions) (*cos.ServiceGetResult, *cos.Response, error) {
			return &cos.ServiceGetResult{
				Buckets: []cos.Bucket{
					{Name: "bucket-a", Region: "ap-guangzhou"},
					{Name: "bucket-b", Region: "ap-beijing"},
				},
				IsTruncated: false,
				NextMarker:  "",
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		buckets, nextMarker, isTruncated, err := GetBucketsList(newTestClient(), 10, "")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if len(buckets) != 2 {
			t.Errorf("期望 2 个桶，实际 %d", len(buckets))
		}
		if isTruncated {
			t.Error("期望 isTruncated=false")
		}
		if nextMarker != "" {
			t.Errorf("期望 nextMarker 为空，实际 %s", nextMarker)
		}
		mockServiceGetFunc = nil
	})

	t.Run("返回分页结果（isTruncated=true）", func(t *testing.T) {
		mockServiceGetFunc = func(ctx context.Context, opt *cos.ServiceGetOptions) (*cos.ServiceGetResult, *cos.Response, error) {
			return &cos.ServiceGetResult{
				Buckets: []cos.Bucket{
					{Name: "bucket-a", Region: "ap-guangzhou"},
				},
				IsTruncated: true,
				NextMarker:  "bucket-a",
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		buckets, nextMarker, isTruncated, err := GetBucketsList(newTestClient(), 1, "")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if len(buckets) != 1 {
			t.Errorf("期望 1 个桶，实际 %d", len(buckets))
		}
		if !isTruncated {
			t.Error("期望 isTruncated=true")
		}
		if nextMarker != "bucket-a" {
			t.Errorf("期望 nextMarker=bucket-a，实际 %s", nextMarker)
		}
		mockServiceGetFunc = nil
	})
}
