package util

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestTryGetUploads(t *testing.T) {
	// tryGetUploads 对非 503 错误直接返回
	opt := &cos.ListMultipartUploadsOptions{Prefix: "prefix/"}

	t.Run("非 503 错误直接返回", func(t *testing.T) {
		mockBucketListMultipartUploadsFunc = func(ctx context.Context, opt *cos.ListMultipartUploadsOptions) (*cos.ListMultipartUploadsResult, *cos.Response, error) {
			return nil, &cos.Response{Response: &http.Response{StatusCode: 403}}, fmt.Errorf("access denied")
		}
		_, err := tryGetUploads(newTestClient(), opt)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		mockBucketListMultipartUploadsFunc = nil
	})

	t.Run("成功时返回结果", func(t *testing.T) {
		mockBucketListMultipartUploadsFunc = func(ctx context.Context, opt *cos.ListMultipartUploadsOptions) (*cos.ListMultipartUploadsResult, *cos.Response, error) {
			return &cos.ListMultipartUploadsResult{
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		res, err := tryGetUploads(newTestClient(), opt)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		if res == nil {
			t.Error("期望返回非 nil 结果")
		}
		mockBucketListMultipartUploadsFunc = nil
	})
}

func TestTryGetParts(t *testing.T) {
	// tryGetParts 对非 503 错误直接返回
	opt := &cos.ObjectListPartsOptions{}

	t.Run("非 503 错误直接返回", func(t *testing.T) {
		mockObjectListPartsFunc = func(ctx context.Context, name, uploadID string, opt *cos.ObjectListPartsOptions) (*cos.ObjectListPartsResult, *cos.Response, error) {
			return nil, &cos.Response{Response: &http.Response{StatusCode: 404}}, fmt.Errorf("not found")
		}
		_, err := tryGetParts(newTestClient(), "file.txt", "upload-id", opt)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		mockObjectListPartsFunc = nil
	})

	t.Run("成功时返回结果", func(t *testing.T) {
		mockObjectListPartsFunc = func(ctx context.Context, name, uploadID string, opt *cos.ObjectListPartsOptions) (*cos.ObjectListPartsResult, *cos.Response, error) {
			return &cos.ObjectListPartsResult{
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		res, err := tryGetParts(newTestClient(), "file.txt", "upload-id", opt)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		if res == nil {
			t.Error("期望返回非 nil 结果")
		}
		mockObjectListPartsFunc = nil
	})
}

func TestDecryptSecretError(t *testing.T) {
	// DecryptSecret 解码 base64 失败返回错误
	t.Run("base64 解码失败", func(t *testing.T) {
		_, err := DecryptSecret("!!!invalid-base64!!!")
		if err == nil {
			t.Error("期望返回错误（base64 解码失败），但得到 nil")
		}
	})

	t.Run("正常加密解密往返", func(t *testing.T) {
		original := "my-secret-key-12345"
		encrypted, err := EncryptSecret(original)
		if err != nil {
			t.Fatalf("加密失败: %v", err)
		}
		decrypted, err := DecryptSecret(encrypted)
		if err != nil {
			t.Fatalf("解密失败: %v", err)
		}
		if decrypted != original {
			t.Errorf("期望 %q，实际 %q", original, decrypted)
		}
	})
}

func TestGetOfsObjectListRecursion(t *testing.T) {
	// getOfsObjectListRecursion 递归扫描 OFS 对象

	t.Run("扫描单层目录无 CommonPrefixes", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents: []cos.Object{
					{Key: "prefix/file1.txt", Size: 100, LastModified: "2024-01-01T00:00:00Z"},
				},
				CommonPrefixes: []string{},
				IsTruncated:    false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}
		chObjects := make(chan objectInfoType, 10)
		chError := make(chan error, 2)
		fo := &FileOperations{
			Operation: Operation{},
			Monitor:   &FileProcessMonitor{},
		}
		go func() {
			getOfsObjectListRecursion(newTestClient(), cosUrl, chObjects, chError, fo, false, "prefix/", "", 0, "/")
			close(chObjects)
		}()

		count := 0
		for range chObjects {
			count++
		}
		mockBucketGetFunc = nil
	})
}
