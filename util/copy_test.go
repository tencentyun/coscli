package util

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// mockObjectMultiCopyFunc 全局 mock 变量，控制 Object.MultiCopy 行为
var mockObjectMultiCopyFunc func(ctx context.Context, key, sourceURL string, opt *cos.MultiCopyOptions, id ...string) (*cos.ObjectCopyResult, *cos.Response, error)

func TestCosCopy(t *testing.T) {
	// CosCopy 内部调用 GetHead（Object.Head 已全局打桩）和 Object.MultiCopy（全局打桩）
	testConfig := &Config{
		Base: BaseCfg{
			SecretID:  "test-secret-id",
			SecretKey: "test-secret-key",
			Protocol:  "https",
		},
		Buckets: []Bucket{
			{
				Name:     "src-bucket-1234567890",
				Alias:    "src-bucket",
				Region:   "ap-guangzhou",
				Endpoint: "cos.ap-guangzhou.myqcloud.com",
			},
			{
				Name:     "dest-bucket-1234567890",
				Alias:    "dest-bucket",
				Region:   "ap-guangzhou",
				Endpoint: "cos.ap-guangzhou.myqcloud.com",
			},
		},
	}
	testParam := &Param{}

	srcUrl := &CosUrl{Bucket: "src-bucket", Object: "src/file.txt"}
	destUrl := &CosUrl{Bucket: "dest-bucket", Object: "dest/file.txt"}

	t.Run("Head 返回 404 时返回 Object not found 错误", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 404}},
				&cos.ErrorResponse{Response: &http.Response{StatusCode: 404}}
		}
		defer func() { mockHeadFunc = nil }()
		fo := &FileOperations{
			Config:    testConfig,
			Param:     testParam,
			CpType:    CpTypeCopy,
			Monitor:   &FileProcessMonitor{},
			ErrOutput: &ErrOutput{},
			Operation: Operation{Routines: 1},
		}
		err := CosCopy(newTestClient(), newTestClient(), srcUrl, destUrl, fo)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("Head 返回其他错误时返回 Head object err", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return nil, fmt.Errorf("mock head error")
		}
		defer func() { mockHeadFunc = nil }()
		fo := &FileOperations{
			Config:    testConfig,
			Param:     testParam,
			CpType:    CpTypeCopy,
			Monitor:   &FileProcessMonitor{},
			ErrOutput: &ErrOutput{},
			Operation: Operation{Routines: 1},
		}
		err := CosCopy(newTestClient(), newTestClient(), srcUrl, destUrl, fo)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("MultiCopy 成功时返回 nil", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{
				StatusCode: 200,
				Header:     http.Header{"Last-Modified": []string{"Mon, 01 Jan 2023 00:00:00 GMT"}},
			}}, nil
		}
		defer func() { mockHeadFunc = nil }()
		mockObjectMultiCopyFunc = func(ctx context.Context, key, sourceURL string, opt *cos.MultiCopyOptions, id ...string) (*cos.ObjectCopyResult, *cos.Response, error) {
			return &cos.ObjectCopyResult{}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		defer func() { mockObjectMultiCopyFunc = nil }()
		fo := &FileOperations{
			Config:        testConfig,
			Param:         testParam,
			CpType:        CpTypeCopy,
			Monitor:       &FileProcessMonitor{},
			ErrOutput:     &ErrOutput{},
			ProcessLogger: &ProcessLogger{},
			Operation:     Operation{Routines: 1, PartSize: 32},
		}
		err := CosCopy(newTestClient(), newTestClient(), srcUrl, destUrl, fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}

// TestNeedCarryVersionId 验证 copy 链路中“是否携带 versionId”的判定逻辑。
// 该判定同时用于：源侧 HEAD/IsExist（按源桶类型）与目标侧 MultiCopy（按目标桶类型）。
// 规则：OFS 桶不接受 versionId；且仅当用户显式指定了 versionId 时才携带。
//
// 说明：gomonkey 在 ARM64 上对“从嵌套函数帧转发的可变参数”存在读取错乱问题，
// 无法可靠断言 MultiCopy/Head 实际收到的 versionId，故将判定逻辑抽取为纯函数
// needCarryVersionId 直接测试，确保四种桶组合 × 是否指定 versionId 的分支决策正确。
func TestNeedCarryVersionId(t *testing.T) {
	cases := []struct {
		name       string
		bucketType string
		versionId  string
		want       bool
	}{
		{"COS 桶且指定 versionId：携带", BucketTypeCos, "v-001", true},
		{"COS 桶但未指定 versionId：不携带", BucketTypeCos, "", false},
		{"OFS 桶且指定 versionId：不携带", BucketTypeOfs, "v-001", false},
		{"OFS 桶且未指定 versionId：不携带", BucketTypeOfs, "", false},
		{"空桶类型且指定 versionId：携带", "", "v-001", true},
		{"空桶类型且未指定 versionId：不携带", "", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := needCarryVersionId(c.bucketType, c.versionId); got != c.want {
				t.Errorf("needCarryVersionId(%q, %q)：期望 %v，实际 %v", c.bucketType, c.versionId, c.want, got)
			}
		})
	}
}

// TestCosCopyVersionIdBucketCombinations 冒烟测试：验证 copy 单对象在四种桶组合下均能
// 走完源侧 HEAD → 目标侧 MultiCopy 流程且不报错，覆盖 needCarryVersionId 在真实调用点
// （CosCopy → GetHead / singleCopy → MultiCopy）的两个分支接入。
// 此处不断言实际携带的 versionId（受 ARM64 gomonkey 可变参数读取限制，
// 判定正确性由 TestNeedCarryVersionId 保证）。
func TestCosCopyVersionIdBucketCombinations(t *testing.T) {
	testConfig := &Config{
		Base: BaseCfg{SecretID: "id", SecretKey: "key", Protocol: "https"},
		Buckets: []Bucket{
			{Name: "src-bucket-1234567890", Alias: "src-bucket", Region: "ap-guangzhou", Endpoint: "cos.ap-guangzhou.myqcloud.com"},
			{Name: "dest-bucket-1234567890", Alias: "dest-bucket", Region: "ap-guangzhou", Endpoint: "cos.ap-guangzhou.myqcloud.com"},
		},
	}
	srcUrl := &CosUrl{Bucket: "src-bucket", Object: "src/file.txt"}
	destUrl := &CosUrl{Bucket: "dest-bucket", Object: "dest/file.txt"}

	mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
		return &cos.Response{Response: &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Last-Modified": []string{"Mon, 01 Jan 2023 00:00:00 GMT"}},
		}}, nil
	}
	mockObjectMultiCopyFunc = func(ctx context.Context, key, sourceURL string, opt *cos.MultiCopyOptions, id ...string) (*cos.ObjectCopyResult, *cos.Response, error) {
		return &cos.ObjectCopyResult{}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
	}
	defer func() {
		mockHeadFunc = nil
		mockObjectMultiCopyFunc = nil
	}()

	newFo := func(srcType, dstType, versionId string) *FileOperations {
		return &FileOperations{
			Config:        testConfig,
			Param:         &Param{},
			CpType:        CpTypeCopy,
			Monitor:       &FileProcessMonitor{},
			ErrOutput:     &ErrOutput{},
			ProcessLogger: &ProcessLogger{},
			BucketType:    srcType,
			DstBucketType: dstType,
			Operation:     Operation{Routines: 1, PartSize: 32, VersionId: versionId},
		}
	}

	cases := []struct {
		name             string
		srcType, dstType string
		versionId        string
	}{
		{"cos→ofs 指定versionId", BucketTypeCos, BucketTypeOfs, "v-001"},
		{"cos→cos 指定versionId", BucketTypeCos, BucketTypeCos, "v-001"},
		{"ofs→cos 未指定versionId", BucketTypeOfs, BucketTypeCos, ""},
		{"ofs→ofs 未指定versionId", BucketTypeOfs, BucketTypeOfs, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := CosCopy(newTestClient(), newTestClient(), srcUrl, destUrl, newFo(c.srcType, c.dstType, c.versionId)); err != nil {
				t.Fatalf("期望无错误，但得到: %v", err)
			}
		})
	}
}
