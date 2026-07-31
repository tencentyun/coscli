package util

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

var mockBucketPutInventoryFunc func(ctx context.Context, id string, opt *cos.BucketPutInventoryOptions) (*cos.Response, error)
var mockBucketGetInventoryFunc func(ctx context.Context, id string) (*cos.BucketGetInventoryResult, *cos.Response, error)
var mockBucketDeleteInventoryFunc func(ctx context.Context, id string) (*cos.Response, error)
var mockBucketPostInventoryFunc func(ctx context.Context, id string, opt *cos.BucketPostInventoryOptions) (*cos.Response, error)
var mockBucketListInventoryFunc func(ctx context.Context, token string) (*cos.ListBucketInventoryConfigResult, *cos.Response, error)

func TestPutBucketInventory(t *testing.T) {
	// Bucket.PutInventory 已在 TestMain 中全局打桩，通过 mockBucketPutInventoryFunc 变量控制行为

	t.Run("configuration 格式错误时返回错误", func(t *testing.T) {
		err := PutBucketInventory(newTestClient(), "inv-001", "not-json-or-xml")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK PutInventory 调用失败", func(t *testing.T) {
		mockBucketPutInventoryFunc = func(ctx context.Context, id string, opt *cos.BucketPutInventoryOptions) (*cos.Response, error) {
			return nil, fmt.Errorf("mock put inventory error")
		}
		xmlContent := `<InventoryConfiguration><Id>inv-001</Id><IsEnabled>true</IsEnabled><IncludedObjectVersions>All</IncludedObjectVersions><Schedule><Frequency>Daily</Frequency></Schedule><Destination><Bucket>qcs::cos:ap-guangzhou:uid/1234567890:examplebucket-1234567890</Bucket><Format>CSV</Format></Destination></InventoryConfiguration>`
		err := PutBucketInventory(newTestClient(), "inv-001", xmlContent)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK PutInventory 调用成功", func(t *testing.T) {
		mockBucketPutInventoryFunc = func(ctx context.Context, id string, opt *cos.BucketPutInventoryOptions) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		xmlContent := `<InventoryConfiguration><Id>inv-001</Id><IsEnabled>true</IsEnabled><IncludedObjectVersions>All</IncludedObjectVersions><Schedule><Frequency>Daily</Frequency></Schedule><Destination><Bucket>qcs::cos:ap-guangzhou:uid/1234567890:examplebucket-1234567890</Bucket><Format>CSV</Format></Destination></InventoryConfiguration>`
		err := PutBucketInventory(newTestClient(), "inv-001", xmlContent)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}

func TestGetBucketInventory(t *testing.T) {
	// Bucket.GetInventory 已在 TestMain 中全局打桩，通过 mockBucketGetInventoryFunc 变量控制行为

	t.Run("SDK GetInventory 调用失败", func(t *testing.T) {
		mockBucketGetInventoryFunc = func(ctx context.Context, id string) (*cos.BucketGetInventoryResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock get inventory error")
		}
		err := GetBucketInventory(newTestClient(), "inv-001")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK GetInventory 调用成功", func(t *testing.T) {
		mockBucketGetInventoryFunc = func(ctx context.Context, id string) (*cos.BucketGetInventoryResult, *cos.Response, error) {
			return &cos.BucketGetInventoryResult{
				ID:                     "inv-001",
				IsEnabled:              "true",
				IncludedObjectVersions: "All",
				Schedule:               &cos.BucketInventorySchedule{Frequency: "Daily"},
				Destination: &cos.BucketInventoryDestination{
					Bucket: "qcs::cos:ap-guangzhou:uid/1234567890:examplebucket-1234567890",
					Format: "CSV",
				},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := GetBucketInventory(newTestClient(), "inv-001")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}

func TestDeleteBucketInventory(t *testing.T) {
	// Bucket.DeleteInventory 已在 TestMain 中全局打桩，通过 mockBucketDeleteInventoryFunc 变量控制行为

	t.Run("SDK DeleteInventory 调用失败", func(t *testing.T) {
		mockBucketDeleteInventoryFunc = func(ctx context.Context, id string) (*cos.Response, error) {
			return nil, fmt.Errorf("mock delete inventory error")
		}
		err := DeleteBucketInventory(newTestClient(), "inv-001")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK DeleteInventory 调用成功", func(t *testing.T) {
		mockBucketDeleteInventoryFunc = func(ctx context.Context, id string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 204}}, nil
		}
		err := DeleteBucketInventory(newTestClient(), "inv-001")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}

func TestListBucketInventory(t *testing.T) {
	// Bucket.ListInventoryConfigurations 已在 TestMain 中全局打桩，通过 mockBucketListInventoryFunc 变量控制行为

	t.Run("SDK ListInventoryConfigurations 调用失败", func(t *testing.T) {
		mockBucketListInventoryFunc = func(ctx context.Context, token string) (*cos.ListBucketInventoryConfigResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock list inventory error")
		}
		err := ListBucketInventory(newTestClient())
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK ListInventoryConfigurations 调用成功（单页）", func(t *testing.T) {
		mockBucketListInventoryFunc = func(ctx context.Context, token string) (*cos.ListBucketInventoryConfigResult, *cos.Response, error) {
			return &cos.ListBucketInventoryConfigResult{
				IsTruncated: false,
				InventoryConfigurations: []cos.BucketListInventoryConfiguartion{
					{
						ID:        "inv-001",
						IsEnabled: "true",
						Schedule:  &cos.BucketInventorySchedule{Frequency: "Daily"},
						Destination: &cos.BucketInventoryDestination{
							Bucket: "qcs::cos:ap-guangzhou:uid/1234567890:examplebucket-1234567890",
							Format: "CSV",
						},
					},
				},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := ListBucketInventory(newTestClient())
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("SDK ListInventoryConfigurations 调用成功（多页）", func(t *testing.T) {
		callCount := 0
		mockBucketListInventoryFunc = func(ctx context.Context, token string) (*cos.ListBucketInventoryConfigResult, *cos.Response, error) {
			callCount++
			if callCount == 1 {
				return &cos.ListBucketInventoryConfigResult{
					IsTruncated:           true,
					NextContinuationToken: "next-token",
					InventoryConfigurations: []cos.BucketListInventoryConfiguartion{
						{
							ID:        "inv-001",
							IsEnabled: "true",
							Schedule:  &cos.BucketInventorySchedule{Frequency: "Daily"},
							Destination: &cos.BucketInventoryDestination{
								Bucket: "qcs::cos:ap-guangzhou:uid/1234567890:examplebucket-1234567890",
								Format: "CSV",
							},
						},
					},
				}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
			}
			return &cos.ListBucketInventoryConfigResult{
				IsTruncated: false,
				InventoryConfigurations: []cos.BucketListInventoryConfiguartion{
					{
						ID:        "inv-002",
						IsEnabled: "false",
						Schedule:  &cos.BucketInventorySchedule{Frequency: "Weekly"},
						Destination: &cos.BucketInventoryDestination{
							Bucket: "qcs::cos:ap-guangzhou:uid/1234567890:examplebucket-1234567890",
							Format: "CSV",
						},
					},
				},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := ListBucketInventory(newTestClient())
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if callCount != 2 {
			t.Errorf("期望调用 2 次，实际 %d 次", callCount)
		}
	})
}

func TestFormatDetailedFields(t *testing.T) {
	t.Run("nil fields 返回 No optional fields", func(t *testing.T) {
		result := formatDetailedFields(nil)
		if result != "No optional fields" {
			t.Errorf("期望 No optional fields，实际: %s", result)
		}
	})

	t.Run("空 fields 返回 No optional fields", func(t *testing.T) {
		result := formatDetailedFields(&cos.BucketInventoryOptionalFields{})
		if result != "No optional fields" {
			t.Errorf("期望 No optional fields，实际: %s", result)
		}
	})

	t.Run("有字段时返回字段列表", func(t *testing.T) {
		result := formatDetailedFields(&cos.BucketInventoryOptionalFields{
			BucketInventoryFields: []string{"Size", "LastModifiedDate", "ETag"},
		})
		if result == "" {
			t.Error("期望 result 不为空")
		}
	})
}

func TestRenderInventoryConfigDetail(t *testing.T) {
	t.Run("完整配置不 panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("RenderInventoryConfigDetail panic: %v", r)
			}
		}()
		config := &cos.BucketGetInventoryResult{
			ID:                     "inv-001",
			IsEnabled:              "true",
			IncludedObjectVersions: "All",
			Schedule:               &cos.BucketInventorySchedule{Frequency: "Daily"},
			Destination: &cos.BucketInventoryDestination{
				Bucket:    "qcs::cos:ap-guangzhou:uid/1234567890:examplebucket-1234567890",
				Format:    "CSV",
				AccountId: "1234567890",
				Prefix:    "inventory/",
			},
			Filter: &cos.BucketInventoryFilter{
				Prefix:       "prefix/",
				StorageClass: "STANDARD",
				Tags:         []cos.ObjectTaggingTag{{Key: "env", Value: "prod"}},
				Period:       &cos.BucketInventoryFilterPeriod{StartTime: 1700000000, EndTime: 1800000000},
			},
			OptionalFields: &cos.BucketInventoryOptionalFields{
				BucketInventoryFields: []string{"Size", "LastModifiedDate"},
			},
		}
		RenderInventoryConfigDetail(config)
	})

	t.Run("nil Schedule 和 Destination 不 panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("RenderInventoryConfigDetail panic: %v", r)
			}
		}()
		config := &cos.BucketGetInventoryResult{
			ID:                     "inv-002",
			IsEnabled:              "false",
			IncludedObjectVersions: "Current",
			Schedule:               nil,
			Destination:            nil,
			Filter:                 nil,
			OptionalFields:         nil,
		}
		RenderInventoryConfigDetail(config)
	})

	t.Run("Filter 有多个 Tags 不 panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("RenderInventoryConfigDetail panic: %v", r)
			}
		}()
		config := &cos.BucketGetInventoryResult{
			ID: "inv-003",
			Filter: &cos.BucketInventoryFilter{
				Tags: []cos.ObjectTaggingTag{
					{Key: "env", Value: "prod"},
					{Key: "team", Value: "backend"},
				},
			},
		}
		RenderInventoryConfigDetail(config)
	})
}

func TestFormatFilter(t *testing.T) {
	t.Run("nil filter 返回 No filter", func(t *testing.T) {
		result := formatFilter(nil)
		if result != "No filter" {
			t.Errorf("期望 No filter，实际: %s", result)
		}
	})

	t.Run("空 filter 返回 No filter", func(t *testing.T) {
		result := formatFilter(&cos.BucketInventoryFilter{})
		if result != "No filter" {
			t.Errorf("期望 No filter，实际: %s", result)
		}
	})

	t.Run("含 Prefix 的 filter", func(t *testing.T) {
		result := formatFilter(&cos.BucketInventoryFilter{Prefix: "test/"})
		if result == "" {
			t.Error("期望 result 不为空")
		}
	})

	t.Run("含 Tags 的 filter", func(t *testing.T) {
		result := formatFilter(&cos.BucketInventoryFilter{
			Tags: []cos.ObjectTaggingTag{{Key: "env", Value: "prod"}},
		})
		if result == "" {
			t.Error("期望 result 不为空")
		}
	})

	t.Run("含 StorageClass 的 filter", func(t *testing.T) {
		result := formatFilter(&cos.BucketInventoryFilter{StorageClass: "STANDARD"})
		if result == "" {
			t.Error("期望 result 不为空")
		}
	})

	t.Run("含 Period 的 filter", func(t *testing.T) {
		result := formatFilter(&cos.BucketInventoryFilter{
			Period: &cos.BucketInventoryFilterPeriod{StartTime: 1700000000, EndTime: 1800000000},
		})
		if result == "" {
			t.Error("期望 result 不为空")
		}
	})
}

func TestPostBucketInventory(t *testing.T) {
	// Bucket.PostInventory 已在 TestMain 中全局打桩，通过 mockBucketPostInventoryFunc 变量控制行为

	t.Run("configuration 格式错误时返回错误", func(t *testing.T) {
		err := PostBucketInventory(newTestClient(), "inv-001", "not-json-or-xml")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK PostInventory 调用失败", func(t *testing.T) {
		mockBucketPostInventoryFunc = func(ctx context.Context, id string, opt *cos.BucketPostInventoryOptions) (*cos.Response, error) {
			return nil, fmt.Errorf("mock post inventory error")
		}
		xmlContent := `<InventoryConfiguration><Id>inv-001</Id><IsEnabled>true</IsEnabled><IncludedObjectVersions>All</IncludedObjectVersions><Schedule><Frequency>Daily</Frequency></Schedule><Destination><Bucket>qcs::cos:ap-guangzhou:uid/1234567890:examplebucket-1234567890</Bucket><Format>CSV</Format></Destination></InventoryConfiguration>`
		err := PostBucketInventory(newTestClient(), "inv-001", xmlContent)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK PostInventory 调用成功", func(t *testing.T) {
		mockBucketPostInventoryFunc = func(ctx context.Context, id string, opt *cos.BucketPostInventoryOptions) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		xmlContent := `<InventoryConfiguration><Id>inv-001</Id><IsEnabled>true</IsEnabled><IncludedObjectVersions>All</IncludedObjectVersions><Schedule><Frequency>Daily</Frequency></Schedule><Destination><Bucket>qcs::cos:ap-guangzhou:uid/1234567890:examplebucket-1234567890</Bucket><Format>CSV</Format></Destination></InventoryConfiguration>`
		err := PostBucketInventory(newTestClient(), "inv-001", xmlContent)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}
