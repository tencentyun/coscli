package util

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

var mockBucketPutACLFunc func(ctx context.Context, opt *cos.BucketPutACLOptions) (*cos.Response, error)
var mockBucketGetACLFunc func(ctx context.Context) (*cos.BucketGetACLResult, *cos.Response, error)
var mockObjectPutACLFunc func(ctx context.Context, name string, opt *cos.ObjectPutACLOptions, id ...string) (*cos.Response, error)
var mockObjectGetACLFunc func(ctx context.Context, name string, id ...string) (*cos.ObjectGetACLResult, *cos.Response, error)

func TestPutBucketAcl(t *testing.T) {
	// Bucket.PutACL 已在 TestMain 中全局打桩，通过 mockBucketPutACLFunc 变量控制行为

	t.Run("SDK PutACL 调用失败", func(t *testing.T) {
		mockBucketPutACLFunc = func(ctx context.Context, opt *cos.BucketPutACLOptions) (*cos.Response, error) {
			return nil, fmt.Errorf("mock put acl error")
		}
		settings := ACLSettings{ACL: "public-read"}
		err := PutBucketAcl(newTestClient(), settings)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK PutACL 调用成功", func(t *testing.T) {
		mockBucketPutACLFunc = func(ctx context.Context, opt *cos.BucketPutACLOptions) (*cos.Response, error) {
			if opt.Header.XCosACL != "public-read" {
				return nil, fmt.Errorf("期望 XCosACL=public-read，实际: %s", opt.Header.XCosACL)
			}
			return &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		settings := ACLSettings{ACL: "public-read"}
		err := PutBucketAcl(newTestClient(), settings)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("SDK PutACL 调用成功（含 GrantRead）", func(t *testing.T) {
		mockBucketPutACLFunc = func(ctx context.Context, opt *cos.BucketPutACLOptions) (*cos.Response, error) {
			if opt.Header.XCosGrantRead != "id=\"100000000001\"" {
				return nil, fmt.Errorf("期望 XCosGrantRead 正确，实际: %s", opt.Header.XCosGrantRead)
			}
			return &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		settings := ACLSettings{GrantRead: "id=\"100000000001\""}
		err := PutBucketAcl(newTestClient(), settings)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}

func TestGetBucketAcl(t *testing.T) {
	// Bucket.GetACL 已在 TestMain 中全局打桩，通过 mockBucketGetACLFunc 变量控制行为

	t.Run("SDK GetACL 调用失败", func(t *testing.T) {
		mockBucketGetACLFunc = func(ctx context.Context) (*cos.BucketGetACLResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock get acl error")
		}
		err := GetBucketAcl(newTestClient())
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK GetACL 调用成功（有 Owner 和 ACL）", func(t *testing.T) {
		mockBucketGetACLFunc = func(ctx context.Context) (*cos.BucketGetACLResult, *cos.Response, error) {
			return &cos.BucketGetACLResult{
				Owner: &cos.Owner{UIN: "100000000001", ID: "qcs::cam::uin/100000000001:uin/100000000001", DisplayName: "test-user"},
				AccessControlList: []cos.ACLGrant{
					{Permission: "FULL_CONTROL", Grantee: &cos.ACLGrantee{Type: "CanonicalUser", ID: "qcs::cam::uin/100000000001:uin/100000000001"}},
				},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := GetBucketAcl(newTestClient())
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("SDK GetACL 调用成功（无 Owner 无 ACL）", func(t *testing.T) {
		mockBucketGetACLFunc = func(ctx context.Context) (*cos.BucketGetACLResult, *cos.Response, error) {
			return &cos.BucketGetACLResult{}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := GetBucketAcl(newTestClient())
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}

func TestPutObjectAcl(t *testing.T) {
	// Object.PutACL 已在 TestMain 中全局打桩，通过 mockObjectPutACLFunc 变量控制行为

	t.Run("SDK PutACL 调用失败", func(t *testing.T) {
		mockObjectPutACLFunc = func(ctx context.Context, name string, opt *cos.ObjectPutACLOptions, id ...string) (*cos.Response, error) {
			return nil, fmt.Errorf("mock put object acl error")
		}
		settings := ACLSettings{ACL: "public-read"}
		err := PutObjectAcl(newTestClient(), "test.txt", "", BucketTypeCos, settings)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK PutACL 调用成功（COS 桶）", func(t *testing.T) {
		mockObjectPutACLFunc = func(ctx context.Context, name string, opt *cos.ObjectPutACLOptions, id ...string) (*cos.Response, error) {
			if opt.Header.XCosACL != "public-read" {
				return nil, fmt.Errorf("期望 XCosACL=public-read，实际: %s", opt.Header.XCosACL)
			}
			return &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		settings := ACLSettings{ACL: "public-read"}
		err := PutObjectAcl(newTestClient(), "test.txt", "", BucketTypeCos, settings)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}

func TestGetObjectAcl(t *testing.T) {
	// Object.GetACL 已在 TestMain 中全局打桩，通过 mockObjectGetACLFunc 变量控制行为

	t.Run("SDK GetACL 调用失败", func(t *testing.T) {
		mockObjectGetACLFunc = func(ctx context.Context, name string, id ...string) (*cos.ObjectGetACLResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock get object acl error")
		}
		err := GetObjectAcl(newTestClient(), "test.txt", "", BucketTypeCos)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK GetACL 调用成功（有 Owner 和 ACL）", func(t *testing.T) {
		mockObjectGetACLFunc = func(ctx context.Context, name string, id ...string) (*cos.ObjectGetACLResult, *cos.Response, error) {
			return &cos.ObjectGetACLResult{
				Owner: &cos.Owner{UIN: "100000000001", DisplayName: "test-user"},
				AccessControlList: []cos.ACLGrant{
					{Permission: "READ", Grantee: &cos.ACLGrantee{Type: "Group", URI: "http://cam.qcloud.com/groups/global/AllUsers"}},
				},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := GetObjectAcl(newTestClient(), "test.txt", "", BucketTypeCos)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("SDK GetACL 调用成功（OFS 桶）", func(t *testing.T) {
		mockObjectGetACLFunc = func(ctx context.Context, name string, id ...string) (*cos.ObjectGetACLResult, *cos.Response, error) {
			return &cos.ObjectGetACLResult{}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := GetObjectAcl(newTestClient(), "test.txt", "", BucketTypeOfs)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}

func TestRenderACLTable(t *testing.T) {
	t.Run("Owner 为 nil 且无授权时不 panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("RenderACLTable panic: %v", r)
			}
		}()
		acl := &cos.ACLXml{
			Owner:             nil,
			AccessControlList: []cos.ACLGrant{},
		}
		RenderACLTable(acl)
	})

	t.Run("有 Owner 信息时不 panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("RenderACLTable panic: %v", r)
			}
		}()
		acl := &cos.ACLXml{
			Owner: &cos.Owner{
				UIN:         "100000000001",
				ID:          "qcs::cam::uin/100000000001:uin/100000000001",
				DisplayName: "test-owner",
			},
			AccessControlList: []cos.ACLGrant{},
		}
		RenderACLTable(acl)
	})

	t.Run("Group 类型授权时不 panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("RenderACLTable panic: %v", r)
			}
		}()
		acl := &cos.ACLXml{
			Owner: &cos.Owner{UIN: "100000000001"},
			AccessControlList: []cos.ACLGrant{
				{
					Permission: "READ",
					Grantee: &cos.ACLGrantee{
						Type: "Group",
						URI:  "http://cam.qcloud.com/groups/global/AllUsers",
					},
				},
			},
		}
		RenderACLTable(acl)
	})

	t.Run("CanonicalUser 类型授权时不 panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("RenderACLTable panic: %v", r)
			}
		}()
		acl := &cos.ACLXml{
			Owner: &cos.Owner{UIN: "100000000001"},
			AccessControlList: []cos.ACLGrant{
				{
					Permission: "FULL_CONTROL",
					Grantee: &cos.ACLGrantee{
						Type:        "CanonicalUser",
						ID:          "qcs::cam::uin/100000000001:uin/100000000001",
						DisplayName: "test-user",
					},
				},
			},
		}
		RenderACLTable(acl)
	})

	t.Run("default 类型授权时不 panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("RenderACLTable panic: %v", r)
			}
		}()
		acl := &cos.ACLXml{
			Owner: &cos.Owner{UIN: "100000000001"},
			AccessControlList: []cos.ACLGrant{
				{
					Permission: "WRITE",
					Grantee: &cos.ACLGrantee{
						Type:       "unknown",
						UIN:        "100000000002",
						SubAccount: "sub-account",
					},
				},
			},
		}
		RenderACLTable(acl)
	})

	t.Run("Grantee 为 nil 时不 panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("RenderACLTable panic: %v", r)
			}
		}()
		acl := &cos.ACLXml{
			Owner: &cos.Owner{UIN: "100000000001"},
			AccessControlList: []cos.ACLGrant{
				{Permission: "READ", Grantee: nil},
			},
		}
		RenderACLTable(acl)
	})

	t.Run("多个授权时不 panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("RenderACLTable panic: %v", r)
			}
		}()
		acl := &cos.ACLXml{
			Owner: &cos.Owner{UIN: "100000000001"},
			AccessControlList: []cos.ACLGrant{
				{Permission: "READ", Grantee: &cos.ACLGrantee{Type: "Group", URI: "http://cam.qcloud.com/groups/global/AllUsers"}},
				{Permission: "WRITE", Grantee: &cos.ACLGrantee{Type: "CanonicalUser", ID: "test-id"}},
			},
		}
		RenderACLTable(acl)
	})
}
