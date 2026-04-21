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
