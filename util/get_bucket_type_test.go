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

var mockBucketHeadFunc func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error)

func TestGetBucketType(t *testing.T) {
	var b *cos.BucketService
	patches := ApplyMethodFunc(reflect.TypeOf(b), "Head",
		func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
			return mockBucketHeadFunc(ctx, opt...)
		})
	defer patches.Reset()

	t.Run("param.BucketType=COS 时直接返回 COS（不调用 SDK）", func(t *testing.T) {
		param := &Param{BucketType: "COS"}
		config := &Config{}
		result, err := GetBucketType(newTestClient(), param, config, "test-bucket")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if result != BucketTypeCos {
			t.Errorf("期望 result=%s，实际: %s", BucketTypeCos, result)
		}
	})

	t.Run("param.BucketType=OFS 时直接返回 OFS（不调用 SDK）", func(t *testing.T) {
		param := &Param{BucketType: "OFS"}
		config := &Config{}
		result, err := GetBucketType(newTestClient(), param, config, "test-bucket")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if result != BucketTypeOfs {
			t.Errorf("期望 result=%s，实际: %s", BucketTypeOfs, result)
		}
	})

	t.Run("DisableAutoFetchBucketType=true 且桶 Ofs=false 时返回 COS", func(t *testing.T) {
		param := &Param{}
		config := &Config{
			Base: BaseCfg{DisableAutoFetchBucketType: "true"},
			Buckets: []Bucket{
				{Name: "test-bucket", Alias: "test-alias", Ofs: false},
			},
		}
		result, err := GetBucketType(newTestClient(), param, config, "test-alias")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if result != BucketTypeCos {
			t.Errorf("期望 result=%s，实际: %s", BucketTypeCos, result)
		}
	})

	t.Run("DisableAutoFetchBucketType=true 且桶 Ofs=true 时返回 OFS", func(t *testing.T) {
		param := &Param{}
		config := &Config{
			Base: BaseCfg{DisableAutoFetchBucketType: "true"},
			Buckets: []Bucket{
				{Name: "test-bucket", Alias: "test-alias", Ofs: true},
			},
		}
		result, err := GetBucketType(newTestClient(), param, config, "test-alias")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if result != BucketTypeOfs {
			t.Errorf("期望 result=%s，实际: %s", BucketTypeOfs, result)
		}
	})

	t.Run("自动获取桶类型时 SDK Head 调用失败", func(t *testing.T) {
		mockBucketHeadFunc = func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
			return nil, fmt.Errorf("mock bucket head error")
		}
		param := &Param{}
		config := &Config{}
		result, err := GetBucketType(newTestClient(), param, config, "test-bucket")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		if result != "" {
			t.Errorf("期望 result 为空，实际: %s", result)
		}
	})

	t.Run("自动获取桶类型时 SDK Head 返回 COS 桶", func(t *testing.T) {
		mockBucketHeadFunc = func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
			h := http.Header{}
			// 不设置 X-Cos-Bucket-Arch，表示 COS 桶
			return &cos.Response{Response: &http.Response{StatusCode: 200, Header: h}}, nil
		}
		param := &Param{}
		config := &Config{}
		result, err := GetBucketType(newTestClient(), param, config, "test-bucket")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if result != BucketTypeCos {
			t.Errorf("期望 result=%s，实际: %s", BucketTypeCos, result)
		}
	})

	t.Run("自动获取桶类型时 SDK Head 返回 OFS 桶", func(t *testing.T) {
		mockBucketHeadFunc = func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
			h := http.Header{}
			h.Set("X-Cos-Bucket-Arch", BucketTypeOfs)
			return &cos.Response{Response: &http.Response{StatusCode: 200, Header: h}}, nil
		}
		param := &Param{}
		config := &Config{}
		result, err := GetBucketType(newTestClient(), param, config, "test-bucket")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if result != BucketTypeOfs {
			t.Errorf("期望 result=%s，实际: %s", BucketTypeOfs, result)
		}
	})
}
