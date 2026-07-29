package util

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestRestoreObjectsCos(t *testing.T) {
	// RestoreObjects (bucketType=Cos) 内部调用 produceCosRestoreTasks → getCosObjectListForLs → Bucket.Get（已全局打桩）
	// 对归档对象通过 worker 池调用 TryRestoreObject → Object.PostRestore（已全局打桩）
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}

	t.Run("无对象时成功返回", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents:    []cos.Object{},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		fo := &FileOperations{
			Operation: Operation{},
			ErrOutput: &ErrOutput{Path: "/tmp"},
		}
		err := RestoreObjects(newTestClient(), cosUrl, fo, BucketTypeCos)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		mockBucketGetFunc = nil
	})

	t.Run("成功对归档对象发起恢复请求", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents: []cos.Object{
					{Key: "prefix/archive-file.txt", StorageClass: Archive},
					{Key: "prefix/standard-file.txt", StorageClass: Standard}, // 非归档，计入 errType
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		var mu sync.Mutex
		var restoredKeys []string
		mockObjectPostRestoreFunc = func(ctx context.Context, name string, opt *cos.ObjectRestoreOptions, id ...string) (*cos.Response, error) {
			mu.Lock()
			restoredKeys = append(restoredKeys, name)
			mu.Unlock()
			return &cos.Response{Response: &http.Response{StatusCode: 202}}, nil
		}
		fo := &FileOperations{
			Operation: Operation{
				Days:        1,
				RestoreMode: "Standard",
			},
			ErrOutput: &ErrOutput{Path: "/tmp"},
		}
		err := RestoreObjects(newTestClient(), cosUrl, fo, BucketTypeCos)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		// 仅归档对象应发起恢复请求，非归档对象被 isRestoreType 过滤
		if len(restoredKeys) != 1 || restoredKeys[0] != "prefix/archive-file.txt" {
			t.Errorf("期望仅恢复 prefix/archive-file.txt，实际 %v", restoredKeys)
		}
		mockBucketGetFunc = nil
		mockObjectPostRestoreFunc = nil
	})

	t.Run("已恢复中的对象不再发起请求", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents: []cos.Object{
					{Key: "prefix/ongoing.txt", StorageClass: Archive, RestoreStatus: "ONGOING"},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		restoreCalled := false
		mockObjectPostRestoreFunc = func(ctx context.Context, name string, opt *cos.ObjectRestoreOptions, id ...string) (*cos.Response, error) {
			restoreCalled = true
			return &cos.Response{Response: &http.Response{StatusCode: 202}}, nil
		}
		fo := &FileOperations{
			Operation: Operation{
				Days:        1,
				RestoreMode: "Standard",
			},
			ErrOutput: &ErrOutput{Path: "/tmp"},
		}
		err := RestoreObjects(newTestClient(), cosUrl, fo, BucketTypeCos)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		// ONGOING 状态的对象不应再发起 PostRestore 请求
		if restoreCalled {
			t.Errorf("ONGOING 状态对象不应再次发起恢复请求")
		}
		mockBucketGetFunc = nil
		mockObjectPostRestoreFunc = nil
	})
}

func TestRestoreObjectsOfs(t *testing.T) {
	// RestoreObjects (bucketType=Ofs) 内部调用 restoreOfsObjects → getOfsObjectListForLs → Bucket.Get（已全局打桩）
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}

	t.Run("OFS 桶成功处理归档对象", func(t *testing.T) {
		callCount := 0
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			callCount++
			if callCount == 1 {
				return &cos.BucketGetResult{
					Contents: []cos.Object{
						{Key: "prefix/ofs-archive.txt", StorageClass: Archive},
					},
					CommonPrefixes: []string{},
					IsTruncated:    false,
				}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
			}
			return &cos.BucketGetResult{IsTruncated: false}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		mockObjectPostRestoreFunc = func(ctx context.Context, name string, opt *cos.ObjectRestoreOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 202}}, nil
		}
		fo := &FileOperations{
			Operation: Operation{
				Days:        1,
				RestoreMode: "Standard",
			},
			ErrOutput: &ErrOutput{Path: "/tmp"},
		}
		err := RestoreObjects(newTestClient(), cosUrl, fo, BucketTypeOfs)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		mockBucketGetFunc = nil
		mockObjectPostRestoreFunc = nil
	})
}

func TestGetOfsKeys(t *testing.T) {
	// GetOfsKeys 内部调用 getOfsObjectList → Bucket.Get（已全局打桩）
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}

	t.Run("成功读取 OFS 对象列表", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents: []cos.Object{
					{Key: "prefix/file1.txt", Size: 100, LastModified: "2024-01-01T00:00:00Z"},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		keys := make(map[string]commonInfoType)
		fo := &FileOperations{
			Operation: Operation{},
			Monitor:   &FileProcessMonitor{},
		}
		err := GetOfsKeys(newTestClient(), cosUrl, keys, fo, TypeSrc)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		mockBucketGetFunc = nil
	})
}

func TestCountOfsObjects(t *testing.T) {
	// countOfsObjects 内部调用 getOfsObjectListForLs → Bucket.Get（已全局打桩）

	t.Run("DU_TYPE_TOTAL 统计所有对象", func(t *testing.T) {
		resetStatisticCounters()
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents: []cos.Object{
					{Key: "prefix/file1.txt", StorageClass: Standard, Size: 1024},
					{Key: "prefix/file2.txt", StorageClass: Standard, Size: 2048},
				},
				CommonPrefixes: []string{},
				IsTruncated:    false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := countOfsObjects(newTestClient(), "prefix/", nil, "", DU_TYPE_TOTAL)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		if totalCnt != 2 {
			t.Errorf("期望 totalCnt=2，实际 %d", totalCnt)
		}
		mockBucketGetFunc = nil
	})
}

func TestListOfsObjects(t *testing.T) {
	// ListOfsObjects 内部调用 getOfsObjects → getOfsObjectListForLs → Bucket.Get（已全局打桩）
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}

	t.Run("成功列出 OFS 对象", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents: []cos.Object{
					{Key: "prefix/file1.txt", Size: 1024, LastModified: "2024-01-01T00:00:00Z", ETag: "etag-001", StorageClass: "STANDARD"},
				},
				CommonPrefixes: []string{},
				IsTruncated:    false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := ListOfsObjects(newTestClient(), cosUrl, 10, false, nil)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		mockBucketGetFunc = nil
	})

	t.Run("列表中含 CommonPrefixes 时递归列出", func(t *testing.T) {
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
		err := ListOfsObjects(newTestClient(), cosUrl, 10, true, nil)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		mockBucketGetFunc = nil
	})
}
