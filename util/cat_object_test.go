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

var mockObjectGetFunc func(ctx context.Context, name string, opt *cos.ObjectGetOptions, id ...string) (*cos.Response, error)

func TestCatObject(t *testing.T) {
	var o *cos.ObjectService
	patches := ApplyMethodFunc(reflect.TypeOf(o), "Get",
		func(ctx context.Context, name string, opt *cos.ObjectGetOptions, id ...string) (*cos.Response, error) {
			return mockObjectGetFunc(ctx, name, opt, id...)
		})
	defer patches.Reset()

	t.Run("SDK Get 调用失败", func(t *testing.T) {
		mockObjectGetFunc = func(ctx context.Context, name string, opt *cos.ObjectGetOptions, id ...string) (*cos.Response, error) {
			return nil, fmt.Errorf("mock get error")
		}
		cosUrl := &CosUrl{Bucket: "test-bucket", Object: "test.txt"}
		err := CatObject(newTestClient(), cosUrl)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK Get 调用成功", func(t *testing.T) {
		mockObjectGetFunc = func(ctx context.Context, name string, opt *cos.ObjectGetOptions, id ...string) (*cos.Response, error) {
			// 验证 ResponseContentType 正确设置
			if opt.ResponseContentType != "text/html" {
				return nil, fmt.Errorf("期望 ResponseContentType=text/html，实际: %s", opt.ResponseContentType)
			}
			// 返回一个空 body 的 Response
			resp := &http.Response{
				StatusCode: 200,
				Header:     http.Header{},
				Body:       http.NoBody,
			}
			return &cos.Response{Response: resp}, nil
		}
		cosUrl := &CosUrl{Bucket: "test-bucket", Object: "test.txt"}
		err := CatObject(newTestClient(), cosUrl)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})
}
