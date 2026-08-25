package util

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestRemoveOfsObjects(t *testing.T) {
	// RemoveOfsObjects 内部调用 getOfsObjectListForLs → Bucket.Get（已全局打桩）
	// 以及 DeleteCosObjects → Object.DeleteMulti（已全局打桩）
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}

	t.Run("成功删除 OFS 对象（无子目录）", func(t *testing.T) {
		callCount := 0
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			callCount++
			if callCount == 1 {
				return &cos.BucketGetResult{
					Contents: []cos.Object{
						{Key: "prefix/file1.txt"},
					},
					CommonPrefixes: []string{},
					IsTruncated:    false,
				}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
			}
			return &cos.BucketGetResult{IsTruncated: false}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		mockDeleteMultiFunc = func(ctx context.Context, opt *cos.ObjectDeleteMultiOptions) (*cos.ObjectDeleteMultiResult, *cos.Response, error) {
			return &cos.ObjectDeleteMultiResult{}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		fo := &FileOperations{
			Operation: Operation{Force: true},
			CpType:    CpTypeCopy,
			Monitor:   &FileProcessMonitor{},
		}
		err := RemoveOfsObjects("", newTestClient(), cosUrl, "prefix/", fo)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		mockBucketGetFunc = nil
		mockDeleteMultiFunc = nil
	})

	t.Run("含 CommonPrefixes 时递归删除", func(t *testing.T) {
		callCount := 0
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			callCount++
			if callCount == 1 {
				return &cos.BucketGetResult{
					Contents:       []cos.Object{},
					CommonPrefixes: []string{"prefix/subdir/"},
					IsTruncated:    false,
				}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
			}
			return &cos.BucketGetResult{IsTruncated: false}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		mockDeleteMultiFunc = func(ctx context.Context, opt *cos.ObjectDeleteMultiOptions) (*cos.ObjectDeleteMultiResult, *cos.Response, error) {
			return &cos.ObjectDeleteMultiResult{}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		fo := &FileOperations{
			Operation: Operation{Force: true},
			CpType:    CpTypeCopy,
			Monitor:   &FileProcessMonitor{},
		}
		err := RemoveOfsObjects("", newTestClient(), cosUrl, "prefix/", fo)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		mockBucketGetFunc = nil
		mockDeleteMultiFunc = nil
	})

	t.Run("DeleteMulti 失败时返回错误", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents:    []cos.Object{{Key: "prefix/file.txt"}},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		mockDeleteMultiFunc = func(ctx context.Context, opt *cos.ObjectDeleteMultiOptions) (*cos.ObjectDeleteMultiResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock delete error")
		}
		fo := &FileOperations{
			Operation: Operation{Force: true},
			CpType:    CpTypeCopy,
			Monitor:   &FileProcessMonitor{},
		}
		err := RemoveOfsObjects("", newTestClient(), cosUrl, "prefix/", fo)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		mockBucketGetFunc = nil
		mockDeleteMultiFunc = nil
	})
}

func TestRemoveOfsObjectsRecursiveEmptyPrefix(t *testing.T) {
	// prefix="" 时遍历 getOfsObjectListForLs 并逐个 Object.Delete

	t.Run("prefix 为空时遍历并删除", func(t *testing.T) {
		callCount := 0
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			callCount++
			if callCount == 1 {
				return &cos.BucketGetResult{
					Contents: []cos.Object{
						{Key: "file1.txt"},
					},
					CommonPrefixes: []string{"subdir/"},
					IsTruncated:    false,
				}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
			}
			return &cos.BucketGetResult{IsTruncated: false}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		mockObjectDeleteFunc = func(ctx context.Context, name string, opt ...*cos.ObjectDeleteOptions) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 204}}, nil
		}
		err := RemoveOfsObjectsRecursive(newTestClient(), "")
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		mockBucketGetFunc = nil
		mockObjectDeleteFunc = nil
	})
}
