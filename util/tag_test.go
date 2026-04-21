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

var mockBucketPutTaggingFunc func(ctx context.Context, opt *cos.BucketPutTaggingOptions) (*cos.Response, error)
var mockBucketGetTaggingFunc func(ctx context.Context) (*cos.BucketGetTaggingResult, *cos.Response, error)
var mockBucketDeleteTaggingFunc func(ctx context.Context) (*cos.Response, error)
var mockObjectPutTaggingFunc func(ctx context.Context, name string, opt *cos.ObjectPutTaggingOptions, id ...string) (*cos.Response, error)
var mockObjectGetTaggingFunc func(ctx context.Context, name string, opt ...interface{}) (*cos.ObjectGetTaggingResult, *cos.Response, error)
var mockObjectDeleteTaggingFunc func(ctx context.Context, name string, opt ...interface{}) (*cos.Response, error)

func TestPutBucketTagging(t *testing.T) {
	var b *cos.BucketService
	patches := ApplyMethodFunc(reflect.TypeOf(b), "PutTagging",
		func(ctx context.Context, opt *cos.BucketPutTaggingOptions) (*cos.Response, error) {
			return mockBucketPutTaggingFunc(ctx, opt)
		})
	defer patches.Reset()

	t.Run("tag 格式错误（缺少 # 分隔符）", func(t *testing.T) {
		err := PutBucketTagging(newTestClient(), []string{"invalid-tag"})
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK PutTagging 调用失败", func(t *testing.T) {
		mockBucketPutTaggingFunc = func(ctx context.Context, opt *cos.BucketPutTaggingOptions) (*cos.Response, error) {
			return nil, fmt.Errorf("mock put tagging error")
		}
		err := PutBucketTagging(newTestClient(), []string{"key1#value1"})
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK PutTagging 调用成功", func(t *testing.T) {
		mockBucketPutTaggingFunc = func(ctx context.Context, opt *cos.BucketPutTaggingOptions) (*cos.Response, error) {
			if len(opt.TagSet) != 2 {
				return nil, fmt.Errorf("期望 TagSet 长度=2，实际: %d", len(opt.TagSet))
			}
			return &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := PutBucketTagging(newTestClient(), []string{"key1#value1", "key2#value2"})
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}

func TestGetBucketTagging(t *testing.T) {
	var b *cos.BucketService
	patches := ApplyMethodFunc(reflect.TypeOf(b), "GetTagging",
		func(ctx context.Context) (*cos.BucketGetTaggingResult, *cos.Response, error) {
			return mockBucketGetTaggingFunc(ctx)
		})
	defer patches.Reset()

	t.Run("SDK GetTagging 调用失败", func(t *testing.T) {
		mockBucketGetTaggingFunc = func(ctx context.Context) (*cos.BucketGetTaggingResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock get tagging error")
		}
		err := GetBucketTagging(newTestClient())
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK GetTagging 调用成功", func(t *testing.T) {
		mockBucketGetTaggingFunc = func(ctx context.Context) (*cos.BucketGetTaggingResult, *cos.Response, error) {
			return &cos.BucketGetTaggingResult{
				TagSet: []cos.BucketTaggingTag{
					{Key: "env", Value: "prod"},
				},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := GetBucketTagging(newTestClient())
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}

func TestDeleteBucketTagging(t *testing.T) {
	var b *cos.BucketService
	patches := ApplyMethodFunc(reflect.TypeOf(b), "DeleteTagging",
		func(ctx context.Context) (*cos.Response, error) {
			return mockBucketDeleteTaggingFunc(ctx)
		})
	defer patches.Reset()

	t.Run("SDK DeleteTagging 调用失败", func(t *testing.T) {
		mockBucketDeleteTaggingFunc = func(ctx context.Context) (*cos.Response, error) {
			return nil, fmt.Errorf("mock delete tagging error")
		}
		err := DeleteBucketTagging(newTestClient())
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK DeleteTagging 调用成功", func(t *testing.T) {
		mockBucketDeleteTaggingFunc = func(ctx context.Context) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 204}}, nil
		}
		err := DeleteBucketTagging(newTestClient())
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}

func TestEncodeTagging(t *testing.T) {
	t.Run("空字符串返回空", func(t *testing.T) {
		result, err := EncodeTagging("")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if result != "" {
			t.Errorf("期望结果为空，实际: %s", result)
		}
	})

	t.Run("单个 key=value 正确编码", func(t *testing.T) {
		result, err := EncodeTagging("key=value")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if result != "key=value" {
			t.Errorf("期望 key=value，实际: %s", result)
		}
	})

	t.Run("多个 key=value 用 & 连接", func(t *testing.T) {
		result, err := EncodeTagging("k1=v1&k2=v2")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if result != "k1=v1&k2=v2" {
			t.Errorf("期望 k1=v1&k2=v2，实际: %s", result)
		}
	})

	t.Run("含特殊字符时进行 URL 编码", func(t *testing.T) {
		result, err := EncodeTagging("key=hello%20world")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if result != "key=hello%2520world" {
			t.Errorf("期望 key=hello%%2520world，实际: %s", result)
		}
	})

	t.Run("空格被去除后正确解析", func(t *testing.T) {
		result, err := EncodeTagging("key = value")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if result != "key=value" {
			t.Errorf("期望 key=value，实际: %s", result)
		}
	})

	t.Run("格式错误（缺少 = 号）时返回错误", func(t *testing.T) {
		_, err := EncodeTagging("invalid-tag-without-equals")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})
}

func TestPutObjectTagging(t *testing.T) {
	var o *cos.ObjectService
	patches := ApplyMethodFunc(reflect.TypeOf(o), "PutTagging",
		func(ctx context.Context, name string, opt *cos.ObjectPutTaggingOptions, id ...string) (*cos.Response, error) {
			return mockObjectPutTaggingFunc(ctx, name, opt, id...)
		})
	defer patches.Reset()

	t.Run("tag 格式错误时返回错误", func(t *testing.T) {
		err := PutObjectTagging(newTestClient(), "test.txt", []string{"invalid"}, "", BucketTypeCos)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("COS 桶 SDK PutTagging 调用失败", func(t *testing.T) {
		mockObjectPutTaggingFunc = func(ctx context.Context, name string, opt *cos.ObjectPutTaggingOptions, id ...string) (*cos.Response, error) {
			return nil, fmt.Errorf("mock put object tagging error")
		}
		err := PutObjectTagging(newTestClient(), "test.txt", []string{"key1#value1"}, "", BucketTypeCos)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("COS 桶 SDK PutTagging 调用成功", func(t *testing.T) {
		mockObjectPutTaggingFunc = func(ctx context.Context, name string, opt *cos.ObjectPutTaggingOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := PutObjectTagging(newTestClient(), "test.txt", []string{"key1#value1"}, "", BucketTypeCos)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("OFS 桶 SDK PutTagging 调用成功", func(t *testing.T) {
		mockObjectPutTaggingFunc = func(ctx context.Context, name string, opt *cos.ObjectPutTaggingOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := PutObjectTagging(newTestClient(), "test.txt", []string{"key1#value1"}, "", BucketTypeOfs)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}

func TestGetObjectTagging(t *testing.T) {
	var o *cos.ObjectService
	patches := ApplyMethodFunc(reflect.TypeOf(o), "GetTagging",
		func(ctx context.Context, name string, opt ...interface{}) (*cos.ObjectGetTaggingResult, *cos.Response, error) {
			return mockObjectGetTaggingFunc(ctx, name, opt...)
		})
	defer patches.Reset()

	t.Run("COS 桶 SDK GetTagging 调用失败", func(t *testing.T) {
		mockObjectGetTaggingFunc = func(ctx context.Context, name string, opt ...interface{}) (*cos.ObjectGetTaggingResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock get object tagging error")
		}
		err := GetObjectTagging(newTestClient(), "test.txt", "", BucketTypeCos)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("COS 桶 SDK GetTagging 调用成功", func(t *testing.T) {
		mockObjectGetTaggingFunc = func(ctx context.Context, name string, opt ...interface{}) (*cos.ObjectGetTaggingResult, *cos.Response, error) {
			return &cos.ObjectGetTaggingResult{
				TagSet: []cos.ObjectTaggingTag{{Key: "env", Value: "test"}},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := GetObjectTagging(newTestClient(), "test.txt", "", BucketTypeCos)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("OFS 桶 SDK GetTagging 调用成功", func(t *testing.T) {
		mockObjectGetTaggingFunc = func(ctx context.Context, name string, opt ...interface{}) (*cos.ObjectGetTaggingResult, *cos.Response, error) {
			return &cos.ObjectGetTaggingResult{}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := GetObjectTagging(newTestClient(), "test.txt", "", BucketTypeOfs)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}

func TestDeleteObjectTagging(t *testing.T) {
	var o *cos.ObjectService
	patches := ApplyMethodFunc(reflect.TypeOf(o), "DeleteTagging",
		func(ctx context.Context, name string, opt ...interface{}) (*cos.Response, error) {
			return mockObjectDeleteTaggingFunc(ctx, name, opt...)
		})
	defer patches.Reset()

	t.Run("COS 桶 SDK DeleteTagging 调用失败", func(t *testing.T) {
		mockObjectDeleteTaggingFunc = func(ctx context.Context, name string, opt ...interface{}) (*cos.Response, error) {
			return nil, fmt.Errorf("mock delete object tagging error")
		}
		err := DeleteObjectTagging(newTestClient(), "test.txt", "", BucketTypeCos)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("COS 桶 SDK DeleteTagging 调用成功", func(t *testing.T) {
		mockObjectDeleteTaggingFunc = func(ctx context.Context, name string, opt ...interface{}) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 204}}, nil
		}
		err := DeleteObjectTagging(newTestClient(), "test.txt", "", BucketTypeCos)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("OFS 桶 SDK DeleteTagging 调用成功", func(t *testing.T) {
		mockObjectDeleteTaggingFunc = func(ctx context.Context, name string, opt ...interface{}) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 204}}, nil
		}
		err := DeleteObjectTagging(newTestClient(), "test.txt", "", BucketTypeOfs)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}

func TestDeleteDesBucketTagging(t *testing.T) {
	var b *cos.BucketService
	patches := ApplyMethodFunc(reflect.TypeOf(b), "GetTagging",
		func(ctx context.Context) (*cos.BucketGetTaggingResult, *cos.Response, error) {
			return mockBucketGetTaggingFunc(ctx)
		})
	patches.ApplyMethodFunc(reflect.TypeOf(b), "PutTagging",
		func(ctx context.Context, opt *cos.BucketPutTaggingOptions) (*cos.Response, error) {
			return mockBucketPutTaggingFunc(ctx, opt)
		})
	defer patches.Reset()

	t.Run("GetTagging 调用失败时返回错误", func(t *testing.T) {
		mockBucketGetTaggingFunc = func(ctx context.Context) (*cos.BucketGetTaggingResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock get tagging error")
		}
		err := DeleteDesBucketTagging(newTestClient(), []string{"key1#value1"})
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("tag 格式错误时返回错误", func(t *testing.T) {
		mockBucketGetTaggingFunc = func(ctx context.Context) (*cos.BucketGetTaggingResult, *cos.Response, error) {
			return &cos.BucketGetTaggingResult{
				TagSet: []cos.BucketTaggingTag{{Key: "key1", Value: "value1"}},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := DeleteDesBucketTagging(newTestClient(), []string{"invalid"})
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("tag 不存在时返回错误", func(t *testing.T) {
		mockBucketGetTaggingFunc = func(ctx context.Context) (*cos.BucketGetTaggingResult, *cos.Response, error) {
			return &cos.BucketGetTaggingResult{
				TagSet: []cos.BucketTaggingTag{{Key: "key1", Value: "value1"}},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := DeleteDesBucketTagging(newTestClient(), []string{"notexist#value"})
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("PutTagging 调用失败时返回错误", func(t *testing.T) {
		mockBucketGetTaggingFunc = func(ctx context.Context) (*cos.BucketGetTaggingResult, *cos.Response, error) {
			return &cos.BucketGetTaggingResult{
				TagSet: []cos.BucketTaggingTag{{Key: "key1", Value: "value1"}, {Key: "key2", Value: "value2"}},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		mockBucketPutTaggingFunc = func(ctx context.Context, opt *cos.BucketPutTaggingOptions) (*cos.Response, error) {
			return nil, fmt.Errorf("mock put tagging error")
		}
		err := DeleteDesBucketTagging(newTestClient(), []string{"key1#value1"})
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("成功删除指定 tag", func(t *testing.T) {
		mockBucketGetTaggingFunc = func(ctx context.Context) (*cos.BucketGetTaggingResult, *cos.Response, error) {
			return &cos.BucketGetTaggingResult{
				TagSet: []cos.BucketTaggingTag{{Key: "key1", Value: "value1"}, {Key: "key2", Value: "value2"}},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		mockBucketPutTaggingFunc = func(ctx context.Context, opt *cos.BucketPutTaggingOptions) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := DeleteDesBucketTagging(newTestClient(), []string{"key1#value1"})
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}

func TestAddBucketTagging(t *testing.T) {
	var b *cos.BucketService
	patches := ApplyMethodFunc(reflect.TypeOf(b), "GetTagging",
		func(ctx context.Context) (*cos.BucketGetTaggingResult, *cos.Response, error) {
			return mockBucketGetTaggingFunc(ctx)
		})
	patches.ApplyMethodFunc(reflect.TypeOf(b), "PutTagging",
		func(ctx context.Context, opt *cos.BucketPutTaggingOptions) (*cos.Response, error) {
			return mockBucketPutTaggingFunc(ctx, opt)
		})
	defer patches.Reset()

	t.Run("GetTagging 调用失败时返回错误", func(t *testing.T) {
		mockBucketGetTaggingFunc = func(ctx context.Context) (*cos.BucketGetTaggingResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock get tagging error")
		}
		err := AddBucketTagging(newTestClient(), []string{"key1#value1"})
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("tag 格式错误时返回错误", func(t *testing.T) {
		mockBucketGetTaggingFunc = func(ctx context.Context) (*cos.BucketGetTaggingResult, *cos.Response, error) {
			return &cos.BucketGetTaggingResult{TagSet: []cos.BucketTaggingTag{}},
				&cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := AddBucketTagging(newTestClient(), []string{"invalid"})
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("tag 已存在时返回错误", func(t *testing.T) {
		mockBucketGetTaggingFunc = func(ctx context.Context) (*cos.BucketGetTaggingResult, *cos.Response, error) {
			return &cos.BucketGetTaggingResult{
				TagSet: []cos.BucketTaggingTag{{Key: "key1", Value: "value1"}},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := AddBucketTagging(newTestClient(), []string{"key1#newvalue"})
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("PutTagging 调用失败时返回错误", func(t *testing.T) {
		mockBucketGetTaggingFunc = func(ctx context.Context) (*cos.BucketGetTaggingResult, *cos.Response, error) {
			return &cos.BucketGetTaggingResult{TagSet: []cos.BucketTaggingTag{}},
				&cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		mockBucketPutTaggingFunc = func(ctx context.Context, opt *cos.BucketPutTaggingOptions) (*cos.Response, error) {
			return nil, fmt.Errorf("mock put tagging error")
		}
		err := AddBucketTagging(newTestClient(), []string{"key1#value1"})
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("成功添加新 tag", func(t *testing.T) {
		mockBucketGetTaggingFunc = func(ctx context.Context) (*cos.BucketGetTaggingResult, *cos.Response, error) {
			return &cos.BucketGetTaggingResult{
				TagSet: []cos.BucketTaggingTag{{Key: "existing", Value: "val"}},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		mockBucketPutTaggingFunc = func(ctx context.Context, opt *cos.BucketPutTaggingOptions) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := AddBucketTagging(newTestClient(), []string{"newkey#newvalue"})
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}

func TestDeleteDesObjectTagging(t *testing.T) {
	var o *cos.ObjectService
	patches := ApplyMethodFunc(reflect.TypeOf(o), "GetTagging",
		func(ctx context.Context, name string, opt ...interface{}) (*cos.ObjectGetTaggingResult, *cos.Response, error) {
			return mockObjectGetTaggingFunc(ctx, name, opt...)
		})
	patches.ApplyMethodFunc(reflect.TypeOf(o), "PutTagging",
		func(ctx context.Context, name string, opt *cos.ObjectPutTaggingOptions, id ...string) (*cos.Response, error) {
			return mockObjectPutTaggingFunc(ctx, name, opt, id...)
		})
	defer patches.Reset()

	t.Run("GetTagging 调用失败时返回错误", func(t *testing.T) {
		mockObjectGetTaggingFunc = func(ctx context.Context, name string, opt ...interface{}) (*cos.ObjectGetTaggingResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock get tagging error")
		}
		err := DeleteDesObjectTagging(newTestClient(), "test.txt", []string{"key1#value1"}, "", BucketTypeCos)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("tag 格式错误时返回错误", func(t *testing.T) {
		mockObjectGetTaggingFunc = func(ctx context.Context, name string, opt ...interface{}) (*cos.ObjectGetTaggingResult, *cos.Response, error) {
			return &cos.ObjectGetTaggingResult{
				TagSet: []cos.ObjectTaggingTag{{Key: "key1", Value: "value1"}},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := DeleteDesObjectTagging(newTestClient(), "test.txt", []string{"invalid"}, "", BucketTypeCos)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("tag 不存在时返回错误", func(t *testing.T) {
		mockObjectGetTaggingFunc = func(ctx context.Context, name string, opt ...interface{}) (*cos.ObjectGetTaggingResult, *cos.Response, error) {
			return &cos.ObjectGetTaggingResult{
				TagSet: []cos.ObjectTaggingTag{{Key: "key1", Value: "value1"}},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := DeleteDesObjectTagging(newTestClient(), "test.txt", []string{"notexist#value"}, "", BucketTypeCos)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("成功删除指定 tag（COS 桶）", func(t *testing.T) {
		mockObjectGetTaggingFunc = func(ctx context.Context, name string, opt ...interface{}) (*cos.ObjectGetTaggingResult, *cos.Response, error) {
			return &cos.ObjectGetTaggingResult{
				TagSet: []cos.ObjectTaggingTag{{Key: "key1", Value: "value1"}, {Key: "key2", Value: "value2"}},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		mockObjectPutTaggingFunc = func(ctx context.Context, name string, opt *cos.ObjectPutTaggingOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := DeleteDesObjectTagging(newTestClient(), "test.txt", []string{"key1#value1"}, "", BucketTypeCos)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("成功删除指定 tag（OFS 桶）", func(t *testing.T) {
		mockObjectGetTaggingFunc = func(ctx context.Context, name string, opt ...interface{}) (*cos.ObjectGetTaggingResult, *cos.Response, error) {
			return &cos.ObjectGetTaggingResult{
				TagSet: []cos.ObjectTaggingTag{{Key: "key1", Value: "value1"}},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		mockObjectPutTaggingFunc = func(ctx context.Context, name string, opt *cos.ObjectPutTaggingOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := DeleteDesObjectTagging(newTestClient(), "test.txt", []string{"key1#value1"}, "", BucketTypeOfs)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}

func TestAddObjectTagging(t *testing.T) {
	var o *cos.ObjectService
	patches := ApplyMethodFunc(reflect.TypeOf(o), "GetTagging",
		func(ctx context.Context, name string, opt ...interface{}) (*cos.ObjectGetTaggingResult, *cos.Response, error) {
			return mockObjectGetTaggingFunc(ctx, name, opt...)
		})
	patches.ApplyMethodFunc(reflect.TypeOf(o), "PutTagging",
		func(ctx context.Context, name string, opt *cos.ObjectPutTaggingOptions, id ...string) (*cos.Response, error) {
			return mockObjectPutTaggingFunc(ctx, name, opt, id...)
		})
	defer patches.Reset()

	t.Run("GetTagging 调用失败时返回错误", func(t *testing.T) {
		mockObjectGetTaggingFunc = func(ctx context.Context, name string, opt ...interface{}) (*cos.ObjectGetTaggingResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock get tagging error")
		}
		err := AddObjectTagging(newTestClient(), "test.txt", []string{"key1#value1"}, "", BucketTypeCos)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("tag 已存在时返回错误", func(t *testing.T) {
		mockObjectGetTaggingFunc = func(ctx context.Context, name string, opt ...interface{}) (*cos.ObjectGetTaggingResult, *cos.Response, error) {
			return &cos.ObjectGetTaggingResult{
				TagSet: []cos.ObjectTaggingTag{{Key: "key1", Value: "value1"}},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := AddObjectTagging(newTestClient(), "test.txt", []string{"key1#newvalue"}, "", BucketTypeCos)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("成功添加新 tag（COS 桶）", func(t *testing.T) {
		mockObjectGetTaggingFunc = func(ctx context.Context, name string, opt ...interface{}) (*cos.ObjectGetTaggingResult, *cos.Response, error) {
			return &cos.ObjectGetTaggingResult{
				TagSet: []cos.ObjectTaggingTag{{Key: "existing", Value: "val"}},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		mockObjectPutTaggingFunc = func(ctx context.Context, name string, opt *cos.ObjectPutTaggingOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := AddObjectTagging(newTestClient(), "test.txt", []string{"newkey#newvalue"}, "", BucketTypeCos)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("成功添加新 tag（OFS 桶）", func(t *testing.T) {
		mockObjectGetTaggingFunc = func(ctx context.Context, name string, opt ...interface{}) (*cos.ObjectGetTaggingResult, *cos.Response, error) {
			return &cos.ObjectGetTaggingResult{TagSet: []cos.ObjectTaggingTag{}},
				&cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		mockObjectPutTaggingFunc = func(ctx context.Context, name string, opt *cos.ObjectPutTaggingOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := AddObjectTagging(newTestClient(), "test.txt", []string{"newkey#newvalue"}, "", BucketTypeOfs)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}
