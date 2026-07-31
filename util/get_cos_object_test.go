package util

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

var mockBucketGetFunc func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error)

func TestCheckCosObjectExist(t *testing.T) {
	// CheckCosObjectExist 内部调用 Object.IsExist，Object.IsExist 内部调用 Object.Head
	// Object.Head 已在 TestMain 中全局打桩，通过 mockHeadFunc 变量控制每个子测试的行为

	t.Run("prefix 为空时直接返回 false", func(t *testing.T) {
		exist, err := CheckCosObjectExist(newTestClient(), "")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if exist {
			t.Error("期望 exist=false")
		}
	})

	t.Run("SDK Head 调用失败（非 404）时返回错误", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return nil, fmt.Errorf("mock head error")
		}
		defer func() { mockHeadFunc = nil }()
		_, err := CheckCosObjectExist(newTestClient(), "test.txt")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("对象存在时返回 true（Head 返回 200）", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
		}
		defer func() { mockHeadFunc = nil }()
		exist, err := CheckCosObjectExist(newTestClient(), "test.txt")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if !exist {
			t.Error("期望 exist=true")
		}
	})

	t.Run("带 versionId 参数时正确传递", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			if len(id) == 0 || id[0] != "version-001" {
				return nil, fmt.Errorf("期望 versionId=version-001，实际: %v", id)
			}
			return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
		}
		defer func() { mockHeadFunc = nil }()
		exist, err := CheckCosObjectExist(newTestClient(), "test.txt", "version-001")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if !exist {
			t.Error("期望 exist=true")
		}
	})
}

func TestCheckCosPathType(t *testing.T) {
	// Bucket.Get 已在 TestMain 中全局打桩，通过 mockBucketGetFunc 变量控制每个子测试的行为
	// tryGetObjects 内部调用 c.Bucket.Get，打桩生效后不会触发重试等待

	fo := &FileOperations{
		Operation:  Operation{},
		BucketType: BucketTypeCos,
	}

	t.Run("prefix 为空时直接返回 isDir=true", func(t *testing.T) {
		isDir, err := CheckCosPathType(newTestClient(), "", 1, fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if !isDir {
			t.Error("期望 isDir=true")
		}
	})

	t.Run("Contents 不为空时返回 isDir=true", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents: []cos.Object{{Key: "prefix/file.txt"}},
			}, nil, nil
		}
		isDir, err := CheckCosPathType(newTestClient(), "prefix/", 1, fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if !isDir {
			t.Error("期望 isDir=true")
		}
	})

	t.Run("Contents 为空时返回 isDir=false（COS 桶）", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents:       []cos.Object{},
				CommonPrefixes: []string{},
			}, nil, nil
		}
		isDir, err := CheckCosPathType(newTestClient(), "prefix/", 1, fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if isDir {
			t.Error("期望 isDir=false")
		}
	})

	t.Run("OFS 桶 CommonPrefixes 不为空时返回 isDir=true", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents:       []cos.Object{},
				CommonPrefixes: []string{"prefix/subdir/"},
			}, nil, nil
		}
		oFo := &FileOperations{
			Operation:  Operation{},
			BucketType: BucketTypeOfs,
		}
		isDir, err := CheckCosPathType(newTestClient(), "prefix/", 1, oFo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if !isDir {
			t.Error("期望 isDir=true（OFS 桶 CommonPrefixes 不为空）")
		}
	})

	t.Run("prefix 不以 / 结尾时自动补充 /", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			// 验证 prefix 已补充 /
			if opt.Prefix != "prefix/" {
				return nil, nil, fmt.Errorf("期望 prefix=prefix/，实际: %s", opt.Prefix)
			}
			return &cos.BucketGetResult{
				Contents: []cos.Object{{Key: "prefix/file.txt"}},
			}, nil, nil
		}
		isDir, err := CheckCosPathType(newTestClient(), "prefix", 1, fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if !isDir {
			t.Error("期望 isDir=true")
		}
	})
}

func TestGetCosObjectListForLs(t *testing.T) {
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}

	t.Run("非递归时 delimiter=/", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			if opt.Delimiter != "/" {
				return nil, nil, fmt.Errorf("期望 delimiter=/，实际: %s", opt.Delimiter)
			}
			return &cos.BucketGetResult{
				Contents:       []cos.Object{{Key: "prefix/file.txt", Size: 1024}},
				CommonPrefixes: []string{"prefix/subdir/"},
				IsTruncated:    false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err, objects, commonPrefixes, isTruncated, _ := getCosObjectListForLs(newTestClient(), cosUrl, "", 10, false)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if len(objects) != 1 {
			t.Errorf("期望 1 个对象，实际: %d", len(objects))
		}
		if len(commonPrefixes) != 1 {
			t.Errorf("期望 1 个 commonPrefix，实际: %d", len(commonPrefixes))
		}
		if isTruncated {
			t.Error("期望 isTruncated=false")
		}
	})

	t.Run("递归时 delimiter 为空", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			if opt.Delimiter != "" {
				return nil, nil, fmt.Errorf("期望 delimiter 为空，实际: %s", opt.Delimiter)
			}
			return &cos.BucketGetResult{
				Contents:    []cos.Object{{Key: "prefix/file.txt", Size: 1024}},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err, objects, _, _, _ := getCosObjectListForLs(newTestClient(), cosUrl, "", 10, true)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if len(objects) != 1 {
			t.Errorf("期望 1 个对象，实际: %d", len(objects))
		}
	})
}

func TestGetCosObjectVersionListForLs(t *testing.T) {
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}

	t.Run("成功返回版本列表和 deleteMarker", func(t *testing.T) {
		mockBucketGetObjectVersionsFunc = func(ctx context.Context, opt *cos.BucketGetObjectVersionsOptions) (*cos.BucketGetObjectVersionsResult, *cos.Response, error) {
			return &cos.BucketGetObjectVersionsResult{
				Version: []cos.ListVersionsResultVersion{
					{Key: "prefix/file.txt", VersionId: "v-001", Size: 1024},
				},
				DeleteMarker: []cos.ListVersionsResultDeleteMarker{
					{Key: "prefix/file2.txt", VersionId: "v-002"},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err, versions, deleteMarkers, _, isTruncated, _, _ := getCosObjectVersionListForLs(newTestClient(), cosUrl, "", "", 10, false)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if len(versions) != 1 {
			t.Errorf("期望 1 个版本，实际: %d", len(versions))
		}
		if len(deleteMarkers) != 1 {
			t.Errorf("期望 1 个 deleteMarker，实际: %d", len(deleteMarkers))
		}
		if isTruncated {
			t.Error("期望 isTruncated=false")
		}
	})
}

func TestCheckDeleteMarkerExist(t *testing.T) {
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}

	t.Run("versionId 存在于 deleteMarker 中时返回 true", func(t *testing.T) {
		mockBucketGetObjectVersionsFunc = func(ctx context.Context, opt *cos.BucketGetObjectVersionsOptions) (*cos.BucketGetObjectVersionsResult, *cos.Response, error) {
			return &cos.BucketGetObjectVersionsResult{
				DeleteMarker: []cos.ListVersionsResultDeleteMarker{
					{Key: "prefix/file.txt", VersionId: "v-target"},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		exist, err := CheckDeleteMarkerExist(newTestClient(), cosUrl, "v-target")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if !exist {
			t.Error("期望 exist=true")
		}
	})

	t.Run("versionId 不存在时返回 false", func(t *testing.T) {
		mockBucketGetObjectVersionsFunc = func(ctx context.Context, opt *cos.BucketGetObjectVersionsOptions) (*cos.BucketGetObjectVersionsResult, *cos.Response, error) {
			return &cos.BucketGetObjectVersionsResult{
				DeleteMarker: []cos.ListVersionsResultDeleteMarker{
					{Key: "prefix/file.txt", VersionId: "v-other"},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		exist, err := CheckDeleteMarkerExist(newTestClient(), cosUrl, "v-target")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if exist {
			t.Error("期望 exist=false")
		}
	})
}
