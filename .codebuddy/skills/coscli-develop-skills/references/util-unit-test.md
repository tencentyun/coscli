
# Skill：util 层单元测试规范

## 概述

`util/` 层的业务函数需要独立的单元测试，与 `cmd/` 层的集成测试互补，共同保证覆盖率达到 95% 以上。

---

## util 层单测 vs cmd 层单测的核心差异

| 对比项 | cmd 层（`cmd/*_test.go`） | util 层（`util/*_test.go`） |
|---|---|---|
| **测试框架** | goconvey + gomonkey | 标准 `testing.T` + gomonkey |
| **测试入口** | `cmd.Execute()`（通过 cobra 调用） | 直接调用 util 函数 |
| **配置文件** | 必须创建临时 `~/.cos.yaml` | **不需要**配置文件 |
| **patches 管理** | 父级 Convey 中声明，`Reset()` 钩子清理 | **单次打桩 + 全局变量控制行为** |
| **断言方式** | `So(e, ShouldBeNil)` | `t.Errorf / t.Fatalf` |
| **子测试结构** | 嵌套 `Convey` 块 | `t.Run(...)` 子测试 |

---

## ARM64 gomonkey 兼容方案（关键）

在 **Apple Silicon（ARM64）** 机器上，gomonkey 对同一个方法**多次独立打桩**（每次打桩后 Reset 再重新打桩）会失败，导致后续测试仍然使用前一个 mock。

**根本原因**：ARM64 的内存页保护机制，`patches.Reset()` 后再次对同一方法打桩时，新的指令替换可能不生效。

### ❌ 错误做法：多次独立打桩同一方法

```go
// 每个子测试独立打桩 → 在 ARM64 上第2个以后的打桩会失败
func TestFoo_Case1(t *testing.T) {
    var o *cos.ObjectService
    patches := ApplyMethodFunc(reflect.TypeOf(o), "Head", func(...) { return nil, errors.New("err") })
    defer patches.Reset()
    // ...
}

func TestFoo_Case2(t *testing.T) {
    var o *cos.ObjectService
    patches := ApplyMethodFunc(reflect.TypeOf(o), "Head", func(...) { return mockResp, nil })
    defer patches.Reset()  // ❌ ARM64 上可能不生效，仍然使用上一个 mock
    // ...
}
```

### ✅ 正确做法：单次打桩 + 全局变量控制行为

```go
// 声明全局 mock 函数变量
var mockHeadFunc func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error)

func TestFoo(t *testing.T) {
    // 只打桩一次，通过 mockHeadFunc 变量控制每个子测试的行为
    var o *cos.ObjectService
    patches := ApplyMethodFunc(reflect.TypeOf(o), "Head",
        func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
            return mockHeadFunc(ctx, name, opt, id...)
        })
    defer patches.Reset()  // 整个 TestFoo 函数结束时统一清理

    t.Run("调用失败", func(t *testing.T) {
        mockHeadFunc = func(...) (*cos.Response, error) {
            return nil, fmt.Errorf("mock error")
        }
        // ...
    })

    t.Run("调用成功", func(t *testing.T) {
        mockHeadFunc = func(...) (*cos.Response, error) {
            return &cos.Response{...}, nil
        }
        // ...
    })
}
```

---

## 测试文件结构模板

```go
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

// newTestClient 构造一个最小可用的 cos.Client（不会真实发起请求，SDK 方法已被打桩）
func newTestClient() *cos.Client {
    return cos.NewClient(nil, nil)
}

// mockXxxFunc 全局 mock 函数变量，通过它控制每个子测试的行为
var mockXxxFunc func(...) (...)

func TestXxxFunction(t *testing.T) {
    // 只打桩一次
    var svc *cos.ObjectService  // 或 *cos.BucketService 等
    patches := ApplyMethodFunc(reflect.TypeOf(svc), "MethodName",
        func(...) (...) {
            return mockXxxFunc(...)
        })
    defer patches.Reset()

    t.Run("SDK 调用失败", func(t *testing.T) {
        mockXxxFunc = func(...) (...) {
            return ..., fmt.Errorf("mock error")
        }
        result, err := XxxFunction(newTestClient(), ...)
        if err == nil {
            t.Errorf("期望返回错误，但得到 nil")
        }
        if result != nil {
            t.Errorf("期望 result 为 nil，但得到 %v", result)
        }
    })

    t.Run("成功路径", func(t *testing.T) {
        mockXxxFunc = func(...) (...) {
            h := http.Header{}
            h.Set("Content-Type", "text/plain")
            return &cos.Response{Response: &http.Response{StatusCode: 200, Header: h}}, nil
        }
        result, err := XxxFunction(newTestClient(), ...)
        if err != nil {
            t.Fatalf("期望无错误，但得到: %v", err)
        }
        if result == nil {
            t.Fatal("期望 result 不为 nil")
        }
        assertEqual(t, "text/plain", result.ContentType, "ContentType")
    })
}

// assertEqual 断言辅助函数（每个 util 测试文件中定义一次）
func assertEqual(t *testing.T, expected, actual, field string) {
    t.Helper()
    if expected != actual {
        t.Errorf("%s: 期望 %q，实际 %q", field, expected, actual)
    }
}
```

---

## 覆盖率要求与检查命令

util 层单测覆盖率必须达到 **95% 以上**，必须覆盖所有主要分支：

| 分支类型 | 测试用例 | 是否必须 |
|---|---|---|
| SDK 调用失败 | 打桩 SDK 方法返回 error | ✅ |
| 成功路径（无可选参数） | 打桩 SDK 方法返回成功 | ✅ |
| 成功路径（有可选参数） | 覆盖带参数的分支（如 versionId） | ✅ |
| 边界条件 | 空字符串、空 map、特殊字符等 | 视情况 |

```bash
# 运行 util 层单测并查看覆盖率
go test -v -gcflags="all=-l" -coverprofile=coverage_util.out ./util/ -run TestXxxFunction
go tool cover -func=coverage_util.out | grep -E "(xxx|total)"
```

---

## 完整示例：util/stat_test.go

以 `StatObject` 函数为例，展示完整的 util 层单测写法：

```go
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

func newTestClient() *cos.Client {
    return cos.NewClient(nil, nil)
}

var mockHeadFunc func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error)

func TestStatObject(t *testing.T) {
    var o *cos.ObjectService
    patches := ApplyMethodFunc(reflect.TypeOf(o), "Head",
        func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
            return mockHeadFunc(ctx, name, opt, id...)
        })
    defer patches.Reset()

    t.Run("SDK Head 调用失败（无 versionId）", func(t *testing.T) {
        mockHeadFunc = func(...) (*cos.Response, error) {
            return nil, fmt.Errorf("mock head error")
        }
        info, err := StatObject(newTestClient(), "test.txt", "")
        if err == nil { t.Errorf("期望返回错误，但得到 nil") }
        if info != nil { t.Errorf("期望 info 为 nil") }
    })

    t.Run("SDK Head 调用失败（有 versionId）", func(t *testing.T) {
        mockHeadFunc = func(...) (*cos.Response, error) {
            return nil, fmt.Errorf("mock head with version error")
        }
        info, err := StatObject(newTestClient(), "test.txt", "v-001")
        if err == nil { t.Errorf("期望返回错误，但得到 nil") }
        if info != nil { t.Errorf("期望 info 为 nil") }
    })

    t.Run("成功获取对象元数据（无 versionId）", func(t *testing.T) {
        mockHeadFunc = func(...) (*cos.Response, error) {
            h := http.Header{}
            h.Set("ETag", `"abc123"`)
            h.Set("Content-Type", "text/plain")
            h.Set("x-cos-storage-class", "STANDARD")
            h.Set("x-cos-meta-author", "test-user")
            return &cos.Response{Response: &http.Response{StatusCode: 200, Header: h}}, nil
        }
        info, err := StatObject(newTestClient(), "test.txt", "")
        if err != nil { t.Fatalf("期望无错误，但得到: %v", err) }
        assertEqual(t, `"abc123"`, info.ETag, "ETag")
        assertEqual(t, "text/plain", info.ContentType, "ContentType")
        assertEqual(t, "STANDARD", info.StorageClass, "StorageClass")
        assertEqual(t, "test-user", info.CustomMeta["x-cos-meta-author"], "CustomMeta[author]")
    })

    t.Run("成功获取对象元数据（有 versionId）", func(t *testing.T) {
        mockHeadFunc = func(...) (*cos.Response, error) {
            h := http.Header{}
            h.Set("ETag", `"def456"`)
            h.Set("x-cos-version-id", "v-001")
            return &cos.Response{Response: &http.Response{StatusCode: 200, Header: h}}, nil
        }
        info, err := StatObject(newTestClient(), "test.json", "v-001")
        if err != nil { t.Fatalf("期望无错误，但得到: %v", err) }
        assertEqual(t, "v-001", info.VersionId, "VersionId")
    })

    t.Run("无自定义元数据时 CustomMeta 为空 map", func(t *testing.T) {
        mockHeadFunc = func(...) (*cos.Response, error) {
            h := http.Header{}
            h.Set("Content-Type", "image/png")
            return &cos.Response{Response: &http.Response{StatusCode: 200, Header: h}}, nil
        }
        info, err := StatObject(newTestClient(), "image.png", "")
        if err != nil { t.Fatalf("期望无错误，但得到: %v", err) }
        if len(info.CustomMeta) != 0 { t.Errorf("期望 CustomMeta 为空") }
    })
}

func assertEqual(t *testing.T, expected, actual, field string) {
    t.Helper()
    if expected != actual {
        t.Errorf("%s: 期望 %q，实际 %q", field, expected, actual)
    }
}
```

---

## 多个 SDK 方法打桩（叠加方式）

当 util 函数内部调用多个 SDK 方法时，使用同一个 `patches` 对象叠加打桩：

```go
var mockPutFunc func(...) (*cos.Response, error)
var mockGetTaggingFunc func(...) (*cos.BucketGetTaggingResult, *cos.Response, error)

func TestSomeFunction(t *testing.T) {
    var obj *cos.ObjectService
    patches := ApplyMethodFunc(reflect.TypeOf(obj), "Put",
        func(...) (*cos.Response, error) { return mockPutFunc(...) })

    var bucket *cos.BucketService
    patches.ApplyMethodFunc(reflect.TypeOf(bucket), "GetTagging",
        func(...) (*cos.BucketGetTaggingResult, *cos.Response, error) { return mockGetTaggingFunc(...) })

    defer patches.Reset()

    t.Run("Put 失败", func(t *testing.T) {
        mockPutFunc = func(...) (*cos.Response, error) { return nil, fmt.Errorf("put error") }
        // ...
    })
    // ...
}
```

---

## 不需要打桩的情况

util 层中**纯逻辑函数**（不调用 SDK、不发起 HTTP 请求）无需打桩，直接测试：

```go
// 纯逻辑函数直接测试，无需 gomonkey
func TestFormatUrl(t *testing.T) {
    t.Run("有效 cos:// URL", func(t *testing.T) {
        url, err := FormatUrl("cos://my-bucket/path/to/file")
        if err != nil { t.Fatalf("期望无错误: %v", err) }
        if !url.IsCosUrl() { t.Error("期望是 COS URL") }
    })

    t.Run("无效 URL", func(t *testing.T) {
        _, err := FormatUrl("invalid-path")
        if err == nil { t.Error("期望返回错误") }
    })
}
```
