package util

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

var mockBucketPutEncryptionFunc func(ctx context.Context, opt *cos.BucketPutEncryptionOptions) (*cos.Response, error)
var mockBucketGetEncryptionFunc func(ctx context.Context) (*cos.BucketGetEncryptionResult, *cos.Response, error)
var mockBucketDeleteEncryptionFunc func(ctx context.Context) (*cos.Response, error)

func TestPutBucketEncryption(t *testing.T) {
	// Bucket.PutEncryption 已在 TestMain 中全局打桩，通过 mockBucketPutEncryptionFunc 变量控制行为

	t.Run("SDK PutEncryption 调用失败", func(t *testing.T) {
		mockBucketPutEncryptionFunc = func(ctx context.Context, opt *cos.BucketPutEncryptionOptions) (*cos.Response, error) {
			return nil, fmt.Errorf("mock put encryption error")
		}
		settings := BucketEncryptionSettings{SSEAlgorithm: "AES256"}
		err := PutBucketEncryption(newTestClient(), settings)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK PutEncryption 调用成功（AES256）", func(t *testing.T) {
		mockBucketPutEncryptionFunc = func(ctx context.Context, opt *cos.BucketPutEncryptionOptions) (*cos.Response, error) {
			if opt.Rule.SSEAlgorithm != "AES256" {
				return nil, fmt.Errorf("期望 SSEAlgorithm=AES256，实际: %s", opt.Rule.SSEAlgorithm)
			}
			return &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		settings := BucketEncryptionSettings{SSEAlgorithm: "AES256"}
		err := PutBucketEncryption(newTestClient(), settings)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("SDK PutEncryption 调用成功（KMS 含 KeyID）", func(t *testing.T) {
		mockBucketPutEncryptionFunc = func(ctx context.Context, opt *cos.BucketPutEncryptionOptions) (*cos.Response, error) {
			if opt.Rule.SSEAlgorithm != "KMS" {
				return nil, fmt.Errorf("期望 SSEAlgorithm=KMS，实际: %s", opt.Rule.SSEAlgorithm)
			}
			if opt.Rule.KMSMasterKeyID != "kms-key-id-001" {
				return nil, fmt.Errorf("期望 KMSMasterKeyID=kms-key-id-001，实际: %s", opt.Rule.KMSMasterKeyID)
			}
			return &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		settings := BucketEncryptionSettings{SSEAlgorithm: "KMS", KMSMasterKeyID: "kms-key-id-001"}
		err := PutBucketEncryption(newTestClient(), settings)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}

func TestGetBucketEncryption(t *testing.T) {
	// Bucket.GetEncryption 已在 TestMain 中全局打桩，通过 mockBucketGetEncryptionFunc 变量控制行为

	t.Run("SDK GetEncryption 调用失败", func(t *testing.T) {
		mockBucketGetEncryptionFunc = func(ctx context.Context) (*cos.BucketGetEncryptionResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock get encryption error")
		}
		err := GetBucketEncryption(newTestClient())
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK GetEncryption 调用成功（AES256）", func(t *testing.T) {
		mockBucketGetEncryptionFunc = func(ctx context.Context) (*cos.BucketGetEncryptionResult, *cos.Response, error) {
			return &cos.BucketGetEncryptionResult{
				Rule: &cos.BucketEncryptionConfiguration{SSEAlgorithm: "AES256"},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := GetBucketEncryption(newTestClient())
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("SDK GetEncryption 调用成功（KMS 含 KeyID）", func(t *testing.T) {
		mockBucketGetEncryptionFunc = func(ctx context.Context) (*cos.BucketGetEncryptionResult, *cos.Response, error) {
			return &cos.BucketGetEncryptionResult{
				Rule: &cos.BucketEncryptionConfiguration{SSEAlgorithm: "KMS", KMSMasterKeyID: "kms-key-id-001"},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := GetBucketEncryption(newTestClient())
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("SDK GetEncryption 调用成功（SM4）", func(t *testing.T) {
		mockBucketGetEncryptionFunc = func(ctx context.Context) (*cos.BucketGetEncryptionResult, *cos.Response, error) {
			return &cos.BucketGetEncryptionResult{
				Rule: &cos.BucketEncryptionConfiguration{SSEAlgorithm: "SM4"},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := GetBucketEncryption(newTestClient())
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("SDK GetEncryption 调用成功（Rule 为 nil）", func(t *testing.T) {
		mockBucketGetEncryptionFunc = func(ctx context.Context) (*cos.BucketGetEncryptionResult, *cos.Response, error) {
			return &cos.BucketGetEncryptionResult{Rule: nil},
				&cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := GetBucketEncryption(newTestClient())
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("SDK GetEncryption 调用成功（未知算法）", func(t *testing.T) {
		mockBucketGetEncryptionFunc = func(ctx context.Context) (*cos.BucketGetEncryptionResult, *cos.Response, error) {
			return &cos.BucketGetEncryptionResult{
				Rule: &cos.BucketEncryptionConfiguration{SSEAlgorithm: "UNKNOWN"},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := GetBucketEncryption(newTestClient())
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}

func TestDeleteBucketEncryption(t *testing.T) {
	// Bucket.DeleteEncryption 已在 TestMain 中全局打桩，通过 mockBucketDeleteEncryptionFunc 变量控制行为

	t.Run("SDK DeleteEncryption 调用失败", func(t *testing.T) {
		mockBucketDeleteEncryptionFunc = func(ctx context.Context) (*cos.Response, error) {
			return nil, fmt.Errorf("mock delete encryption error")
		}
		err := DeleteBucketEncryption(newTestClient())
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK DeleteEncryption 调用成功", func(t *testing.T) {
		mockBucketDeleteEncryptionFunc = func(ctx context.Context) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 204}}, nil
		}
		err := DeleteBucketEncryption(newTestClient())
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}
