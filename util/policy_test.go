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

var mockBucketPutPolicyFunc func(ctx context.Context, opt *cos.BucketPutPolicyOptions) (*cos.Response, error)
var mockBucketGetPolicyFunc func(ctx context.Context) (*cos.BucketGetPolicyResult, *cos.Response, error)
var mockBucketDeletePolicyFunc func(ctx context.Context) (*cos.Response, error)

func TestPutBucketPolicy(t *testing.T) {
	var b *cos.BucketService
	patches := ApplyMethodFunc(reflect.TypeOf(b), "PutPolicy",
		func(ctx context.Context, opt *cos.BucketPutPolicyOptions) (*cos.Response, error) {
			return mockBucketPutPolicyFunc(ctx, opt)
		})
	defer patches.Reset()

	t.Run("policy 内容格式错误（非 JSON）时返回错误", func(t *testing.T) {
		err := PutBucketPolicy(newTestClient(), "not-json-content")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK PutPolicy 调用失败", func(t *testing.T) {
		mockBucketPutPolicyFunc = func(ctx context.Context, opt *cos.BucketPutPolicyOptions) (*cos.Response, error) {
			return nil, fmt.Errorf("mock put policy error")
		}
		policy := `{"version":"2.0","statement":[]}`
		err := PutBucketPolicy(newTestClient(), policy)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK PutPolicy 调用成功", func(t *testing.T) {
		mockBucketPutPolicyFunc = func(ctx context.Context, opt *cos.BucketPutPolicyOptions) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		policy := `{"version":"2.0","statement":[]}`
		err := PutBucketPolicy(newTestClient(), policy)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}

func TestGetBucketPolicy(t *testing.T) {
	var b *cos.BucketService
	patches := ApplyMethodFunc(reflect.TypeOf(b), "GetPolicy",
		func(ctx context.Context) (*cos.BucketGetPolicyResult, *cos.Response, error) {
			return mockBucketGetPolicyFunc(ctx)
		})
	defer patches.Reset()

	t.Run("SDK GetPolicy 调用失败", func(t *testing.T) {
		mockBucketGetPolicyFunc = func(ctx context.Context) (*cos.BucketGetPolicyResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock get policy error")
		}
		err := GetBucketPolicy(newTestClient())
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK GetPolicy 调用成功（空 Statement）", func(t *testing.T) {
		mockBucketGetPolicyFunc = func(ctx context.Context) (*cos.BucketGetPolicyResult, *cos.Response, error) {
			return &cos.BucketGetPolicyResult{
				Version:   "2.0",
				Statement: []cos.BucketStatement{},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := GetBucketPolicy(newTestClient())
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("SDK GetPolicy 调用成功（含 Statement）", func(t *testing.T) {
		mockBucketGetPolicyFunc = func(ctx context.Context) (*cos.BucketGetPolicyResult, *cos.Response, error) {
			return &cos.BucketGetPolicyResult{
				Version: "2.0",
				Statement: []cos.BucketStatement{
					{
						Sid:    "stmt-1",
						Effect: "Allow",
						Principal: map[string][]string{
							"qcs": {"qcs::cam::uin/100000000001:uin/100000000001"},
						},
						Action:   []string{"name/cos:GetObject"},
						Resource: []string{"qcs::cos:ap-guangzhou:uid/1234567890:examplebucket-1234567890/*"},
					},
				},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := GetBucketPolicy(newTestClient())
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}

func TestDeleteBucketPolicy(t *testing.T) {
	var b *cos.BucketService
	patches := ApplyMethodFunc(reflect.TypeOf(b), "DeletePolicy",
		func(ctx context.Context) (*cos.Response, error) {
			return mockBucketDeletePolicyFunc(ctx)
		})
	defer patches.Reset()

	t.Run("SDK DeletePolicy 调用失败", func(t *testing.T) {
		mockBucketDeletePolicyFunc = func(ctx context.Context) (*cos.Response, error) {
			return nil, fmt.Errorf("mock delete policy error")
		}
		err := DeleteBucketPolicy(newTestClient())
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK DeletePolicy 调用成功", func(t *testing.T) {
		mockBucketDeletePolicyFunc = func(ctx context.Context) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 204}}, nil
		}
		err := DeleteBucketPolicy(newTestClient())
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}

func TestFormatPrincipal(t *testing.T) {
	t.Run("单个 principal 格式化", func(t *testing.T) {
		principal := map[string][]string{
			"qcs": {"qcs::cam::uin/100000000001:uin/100000000001"},
		}
		result := formatPrincipal(principal)
		if result == "" {
			t.Error("期望 result 不为空")
		}
	})

	t.Run("空 principal 格式化", func(t *testing.T) {
		result := formatPrincipal(map[string][]string{})
		if result != "" {
			t.Errorf("期望 result 为空，实际: %s", result)
		}
	})
}

func TestFormatCondition(t *testing.T) {
	t.Run("string 类型 condition 格式化", func(t *testing.T) {
		condition := map[string]map[string]interface{}{
			"StringEquals": {
				"cos:prefix": "test/",
			},
		}
		result := formatCondition(condition)
		if result == "" {
			t.Error("期望 result 不为空")
		}
	})

	t.Run("[]interface{} 类型 condition 格式化", func(t *testing.T) {
		condition := map[string]map[string]interface{}{
			"StringEquals": {
				"cos:prefix": []interface{}{"test/", "prod/"},
			},
		}
		result := formatCondition(condition)
		if result == "" {
			t.Error("期望 result 不为空")
		}
	})

	t.Run("其他类型 condition 格式化", func(t *testing.T) {
		condition := map[string]map[string]interface{}{
			"NumericLessThan": {
				"cos:content-length": 1024,
			},
		}
		result := formatCondition(condition)
		if result == "" {
			t.Error("期望 result 不为空")
		}
	})
}
