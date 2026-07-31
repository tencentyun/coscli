package util

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestGetObjectsListIterator(t *testing.T) {
	// Bucket.Get 已在 TestMain 中全局打桩，通过 mockBucketGetFunc 控制行为

	t.Run("Bucket.Get 失败时返回错误", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock bucket get error")
		}
		_, _, _, _, err := GetObjectsListIterator(newTestClient(), "prefix/", "", "", "")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		mockBucketGetFunc = nil
	})

	t.Run("成功返回对象列表（无 include/exclude）", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents: []cos.Object{
					{Key: "prefix/file1.txt"},
					{Key: "prefix/file2.jpg"},
				},
				CommonPrefixes: []string{"prefix/subdir/"},
				IsTruncated:    false,
				NextMarker:     "",
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		objects, isTruncated, nextMarker, commonPrefixes, err := GetObjectsListIterator(newTestClient(), "prefix/", "", "", "")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if len(objects) != 2 {
			t.Errorf("期望 2 个对象，实际 %d", len(objects))
		}
		if isTruncated {
			t.Error("期望 isTruncated=false")
		}
		if nextMarker != "" {
			t.Errorf("期望 nextMarker 为空，实际 %s", nextMarker)
		}
		if len(commonPrefixes) != 1 {
			t.Errorf("期望 1 个 commonPrefix，实际 %d", len(commonPrefixes))
		}
		mockBucketGetFunc = nil
	})

	t.Run("include 过滤只返回匹配的对象", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents: []cos.Object{
					{Key: "prefix/file1.txt"},
					{Key: "prefix/file2.jpg"},
					{Key: "prefix/file3.txt"},
				},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		objects, _, _, _, err := GetObjectsListIterator(newTestClient(), "prefix/", "", `\.txt$`, "")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if len(objects) != 2 {
			t.Errorf("期望 2 个 txt 对象，实际 %d", len(objects))
		}
		mockBucketGetFunc = nil
	})

	t.Run("exclude 过滤排除匹配的对象", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents: []cos.Object{
					{Key: "prefix/file1.txt"},
					{Key: "prefix/file2.jpg"},
					{Key: "prefix/file3.txt"},
				},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		objects, _, _, _, err := GetObjectsListIterator(newTestClient(), "prefix/", "", "", `\.jpg$`)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if len(objects) != 2 {
			t.Errorf("期望 2 个非 jpg 对象，实际 %d", len(objects))
		}
		mockBucketGetFunc = nil
	})
}
