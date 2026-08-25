package util

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// ---- versioning ----

var mockBucketGetVersioningFunc func(ctx context.Context) (*cos.BucketGetVersionResult, *cos.Response, error)
var mockBucketPutVersioningFunc func(ctx context.Context, opt *cos.BucketPutVersionOptions) (*cos.Response, error)

func TestGetBucketVersioning(t *testing.T) {
	// Bucket.GetVersioning 已在 TestMain 中全局打桩，通过 mockBucketGetVersioningFunc 变量控制行为

	t.Run("SDK GetVersioning 调用失败", func(t *testing.T) {
		mockBucketGetVersioningFunc = func(ctx context.Context) (*cos.BucketGetVersionResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock get versioning error")
		}
		res, resp, err := GetBucketVersioning(newTestClient())
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		if res != nil {
			t.Errorf("期望 res 为 nil，但得到 %v", res)
		}
		if resp != nil {
			t.Errorf("期望 resp 为 nil，但得到 %v", resp)
		}
	})

	t.Run("SDK GetVersioning 调用成功", func(t *testing.T) {
		mockBucketGetVersioningFunc = func(ctx context.Context) (*cos.BucketGetVersionResult, *cos.Response, error) {
			return &cos.BucketGetVersionResult{Status: "Enabled"},
				&cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		res, resp, err := GetBucketVersioning(newTestClient())
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if res == nil {
			t.Fatal("期望 res 不为 nil")
		}
		if res.Status != "Enabled" {
			t.Errorf("期望 Status=Enabled，实际: %s", res.Status)
		}
		if resp == nil {
			t.Error("期望 resp 不为 nil")
		}
	})
}

func TestPutBucketVersioning(t *testing.T) {
	// Bucket.PutVersioning 已在 TestMain 中全局打桩，通过 mockBucketPutVersioningFunc 变量控制行为

	t.Run("SDK PutVersioning 调用失败", func(t *testing.T) {
		mockBucketPutVersioningFunc = func(ctx context.Context, opt *cos.BucketPutVersionOptions) (*cos.Response, error) {
			return nil, fmt.Errorf("mock put versioning error")
		}
		resp, err := PutBucketVersioning(newTestClient(), "Enabled")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		if resp != nil {
			t.Errorf("期望 resp 为 nil，但得到 %v", resp)
		}
	})

	t.Run("SDK PutVersioning 调用成功（Enabled）", func(t *testing.T) {
		mockBucketPutVersioningFunc = func(ctx context.Context, opt *cos.BucketPutVersionOptions) (*cos.Response, error) {
			if opt.Status != "Enabled" {
				return nil, fmt.Errorf("期望 Status=Enabled，实际: %s", opt.Status)
			}
			return &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		resp, err := PutBucketVersioning(newTestClient(), "Enabled")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if resp == nil {
			t.Error("期望 resp 不为 nil")
		}
	})

	t.Run("SDK PutVersioning 调用成功（Suspended）", func(t *testing.T) {
		mockBucketPutVersioningFunc = func(ctx context.Context, opt *cos.BucketPutVersionOptions) (*cos.Response, error) {
			if opt.Status != "Suspended" {
				return nil, fmt.Errorf("期望 Status=Suspended，实际: %s", opt.Status)
			}
			return &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		resp, err := PutBucketVersioning(newTestClient(), "Suspended")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if resp == nil {
			t.Error("期望 resp 不为 nil")
		}
	})
}
