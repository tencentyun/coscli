package util

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	"github.com/tencentyun/cos-go-sdk-v5"
)

var mockObjectPostRestoreFunc func(ctx context.Context, name string, opt *cos.ObjectRestoreOptions, id ...string) (*cos.Response, error)

func TestTryRestoreObject(t *testing.T) {
	var o *cos.ObjectService
	patches := ApplyMethodFunc(reflect.TypeOf(o), "PostRestore",
		func(ctx context.Context, name string, opt *cos.ObjectRestoreOptions, id ...string) (*cos.Response, error) {
			return mockObjectPostRestoreFunc(ctx, name, opt, id...)
		})
	defer patches.Reset()

	t.Run("SDK PostRestore 调用成功", func(t *testing.T) {
		mockObjectPostRestoreFunc = func(ctx context.Context, name string, opt *cos.ObjectRestoreOptions, id ...string) (*cos.Response, error) {
			if opt.Days != 7 {
				return nil, fmt.Errorf("期望 Days=7，实际: %d", opt.Days)
			}
			if opt.Tier.Tier != "Standard" {
				return nil, fmt.Errorf("期望 Tier=Standard，实际: %s", opt.Tier.Tier)
			}
			return &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		resp, err := TryRestoreObject(newTestClient(), "test-bucket", "archive/file.txt", 7, "Standard")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if resp == nil {
			t.Error("期望 resp 不为 nil")
		}
	})

	t.Run("SDK PostRestore 调用失败（非 503）", func(t *testing.T) {
		mockObjectPostRestoreFunc = func(ctx context.Context, name string, opt *cos.ObjectRestoreOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 400}},
				fmt.Errorf("mock restore error")
		}
		resp, err := TryRestoreObject(newTestClient(), "test-bucket", "archive/file.txt", 7, "Standard")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		if resp == nil {
			t.Error("期望 resp 不为 nil（即使失败也应返回 resp）")
		}
	})

	t.Run("SDK PostRestore 返回 409（已在回热中）", func(t *testing.T) {
		mockObjectPostRestoreFunc = func(ctx context.Context, name string, opt *cos.ObjectRestoreOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 409}},
				fmt.Errorf("conflict: already restoring")
		}
		resp, err := TryRestoreObject(newTestClient(), "test-bucket", "archive/file.txt", 7, "Standard")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		if resp == nil {
			t.Error("期望 resp 不为 nil")
		}
		if resp.StatusCode != 409 {
			t.Errorf("期望 StatusCode=409，实际: %d", resp.StatusCode)
		}
	})
}

func TestIsRestoreType(t *testing.T) {
	tests := []struct {
		name         string
		storageClass string
		storageTier  string
		expected     bool
	}{
		{"Archive 类型", Archive, "", true},
		{"MAZArchive 类型", MAZArchive, "", true},
		{"DeepArchive 类型", DeepArchive, "", true},
		{"IntelligentTiering 归档层", IntelligentTiering, StorageTierArchive, true},
		{"IntelligentTiering 深度归档层", IntelligentTiering, StorageTierDeepArchive, true},
		{"IntelligentTiering 标准层（不需要回热）", IntelligentTiering, "STANDARD", false},
		{"MAZIntelligentTiering 归档层", MAZIntelligentTiering, StorageTierArchive, true},
		{"Standard 类型（不需要回热）", Standard, "", false},
		{"StandardIA 类型（不需要回热）", StandardIA, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := cos.Object{
				StorageClass: tt.storageClass,
				StorageTier:  tt.storageTier,
			}
			result := isRestoreType(obj)
			if result != tt.expected {
				t.Errorf("isRestoreType(%s, %s): 期望 %v，实际 %v",
					tt.storageClass, tt.storageTier, tt.expected, result)
			}
		})
	}
}
