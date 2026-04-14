# 规则：测试规范

## 测试框架

```go
import (
    . "github.com/agiledragon/gomonkey/v2"  // 函数打桩
    . "github.com/smartystreets/goconvey/convey"  // BDD 断言
)
```

## 测试文件组织

- 测试文件与命令文件同包（`package cmd`），文件名为 `<command>_test.go`
- 公共测试基础设施在 `cmd/testconfig_test.go` 中定义

## 覆盖率要求

**单测覆盖率必须达到 95% 以上**，包括：

- `cmd/<command>.go` 中 `RunE` 函数的所有分支（参数校验、Client 创建失败、业务函数失败、成功路径）
- `util/<command>.go` 中所有业务函数的主要分支

### 覆盖率检查命令

```bash
# 运行单个命令的测试并查看覆盖率
go test -v -gcflags="all=-l" -coverprofile=coverage.out ./cmd/ -run TestXxxCmd
go tool cover -func=coverage.out | grep -E "(xxx|total)"

# 查看覆盖率 HTML 报告
go tool cover -html=coverage.out -o coverage.html
```

### 覆盖率达标清单

每个命令的测试用例必须覆盖以下所有分支：

| 分支类型 | 测试用例 | 是否必须 |
|---|---|---|
| 参数数量不足 | 不传参数或参数不够 | ✅ |
| URL 格式错误 | 传入非 `cos://` 路径 | ✅ |
| SDK 调用失败 | 打桩 cos SDK 方法返回 error | ✅ |
| 成功路径 | 打桩 cos SDK 方法返回成功 | ✅ |
| 各 Flag 组合 | 覆盖所有重要 Flag | ✅ |
| 边界条件 | 空字符串、特殊字符等 | 视情况 |

## goconvey 执行机制说明

goconvey 采用**深度优先树形遍历**执行，每次只执行一条从根到叶的完整路径：

```
第1次执行：根 → success → 0 success 0 fail
第2次执行：根 → success → 1 success
第3次执行：根 → failed → not enough argument
...
```

每次执行都会**重新进入所有父级 Convey 块**。因此：

- **`defer patches.Reset()` 在嵌套 Convey 中不可靠**：`defer` 是函数级别的，不是 Convey 块级别的，会在整个 `TestXxxCmd` 函数返回时才执行，导致多个子用例的桩叠加，行为不可预期。
- **手动在每个子用例写 `clearCmd()` 容易遗漏**，且顺序不统一。

**正确做法：使用 `Reset()` 钩子统一管理**，`Reset()` 在每条路径执行完毕后自动调用，等价于 xUnit 的 `AfterEach`，完全契合 goconvey 的树形遍历机制。

## 测试函数结构

**所有测试用例中只对直接调用 cos go SDK 的方法打桩，禁止在单测中产生真实的外部服务调用。util 层的方法（包括 `util.NewClient`、`util.Upload` 等）不需要打桩，让它们正常执行。**

测试用例必须覆盖命令的**所有执行分支**，确保覆盖率达到 95% 以上。

```go
func TestXxxCmd(t *testing.T) {
    setupTestConfig()           // 创建临时配置文件
    defer teardownTestConfig()  // 测试结束后删除

    Convey("Test coscli xxx", t, func() {
        // ✅ 用 Reset() 统一管理 patches 清理和 clearCmd()，替代每个子用例手动写
        // Reset() 在每条路径执行完毕后自动调用，行为可预期
        var patches *Patches
        Reset(func() {
            if patches != nil {
                patches.Reset()
                patches = nil
            }
            clearCmd()
        })

        // ① 参数数量不足（无需打桩）
        Convey("参数不足", func() {
            cmd := rootCmd
            cmd.SetArgs([]string{"xxx"})
            e := cmd.Execute()
            So(e, ShouldBeError)
        })
        // ② URL 格式错误（无需打桩）
        Convey("URL 格式错误", func() {
            cmd := rootCmd
            cmd.SetArgs([]string{"xxx", "invalid-path"})
            e := cmd.Execute()
            So(e, ShouldBeError)
        })
        // ③ SDK 调用失败（只打桩 cos SDK 方法）
        Convey("SDK 调用失败", func() {
            var obj *cos.ObjectService
            patches = ApplyMethodFunc(reflect.TypeOf(obj), "Put",
                func(ctx context.Context, name string, r io.Reader, opt *cos.ObjectPutOptions) (*cos.Response, error) {
                    return nil, fmt.Errorf("mock sdk error")
                })
            cmd := rootCmd
            cmd.SetArgs([]string{"xxx", "cos://bucket/key"})
            e := cmd.Execute()
            So(e, ShouldBeError)
        })
        // ④ 成功路径（只打桩 cos SDK 方法）
        Convey("成功路径", func() {
            var obj *cos.ObjectService
            patches = ApplyMethodFunc(reflect.TypeOf(obj), "Put",
                func(ctx context.Context, name string, r io.Reader, opt *cos.ObjectPutOptions) (*cos.Response, error) {
                    return &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
                })
            cmd := rootCmd
            cmd.SetArgs([]string{"xxx", "cos://bucket/key", "--flag", "value"})
            e := cmd.Execute()
            So(e, ShouldBeNil)
        })
        // ⑤ 重要 Flag 组合（覆盖各 Flag 分支）
        Convey("带 --flag 参数的成功路径", func() {
            var obj *cos.ObjectService
            patches = ApplyMethodFunc(reflect.TypeOf(obj), "Put",
                func(ctx context.Context, name string, r io.Reader, opt *cos.ObjectPutOptions) (*cos.Response, error) {
                    return &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
                })
            cmd := rootCmd
            cmd.SetArgs([]string{"xxx", "cos://bucket/key", "--flag", "value"})
            e := cmd.Execute()
            So(e, ShouldBeNil)
        })
    })
}
```

## clearCmd() 规范

**`clearCmd()` 通过 `Reset()` 钩子在每条路径执行完毕后自动调用**，无需在每个子用例中手动写：

```go
func clearCmd() {
    rootCmd.Flags().VisitAll(func(flag *pflag.Flag) {
        flag.Value.Set(flag.DefValue)
    })
    for _, subCmd := range rootCmd.Commands() {
        subCmd.Flags().VisitAll(func(flag *pflag.Flag) {
            flag.Value.Set(flag.DefValue)
        })
    }
}
```

## gomonkey 打桩规范

**打桩边界原则：只对直接调用 cos go SDK 的方法打桩，util 层的所有方法（包括 `util.NewClient`、`util.Upload`、`util.GetBucketType` 等）均不需要打桩，让它们正常执行。**

**打桩变量必须声明在父级 Convey 块中**（如 `var patches *Patches`），通过 `Reset()` 钩子统一清理，禁止在子用例中使用 `defer patches.Reset()`。

```go
// ✅ 正确：在父级 Convey 中声明 patches，通过 Reset() 统一清理
Convey("Test coscli xxx", t, func() {
    var patches *Patches
    Reset(func() {
        if patches != nil {
            patches.Reset()
            patches = nil
        }
        clearCmd()
    })

    Convey("SDK 调用失败", func() {
        // 打桩 cos SDK Object 方法（使用 reflect.TypeOf + ApplyMethodFunc）
        var obj *cos.ObjectService
        patches = ApplyMethodFunc(reflect.TypeOf(obj), "Put",
            func(ctx context.Context, name string, r io.Reader, opt *cos.ObjectPutOptions) (*cos.Response, error) {
                return nil, fmt.Errorf("mock sdk error")
            })
        // ...
    })

    Convey("多个 SDK 方法打桩叠加", func() {
        // 使用同一个 patches 对象叠加多个打桩
        var obj *cos.ObjectService
        patches = ApplyMethodFunc(reflect.TypeOf(obj), "Put", func(...) (*cos.Response, error) {
            return &cos.Response{}, nil
        })
        var bucket *cos.BucketService
        patches.ApplyMethodFunc(reflect.TypeOf(bucket), "GetObjectVersions", func(...) (*cos.BucketGetObjectVersionsResult, *cos.Response, error) {
            return &cos.BucketGetObjectVersionsResult{}, &cos.Response{}, nil
        })
        // ...
    })
})

// ❌ 错误：在子用例中使用 defer patches.Reset()（在 goconvey 嵌套中不可靠）
// Convey("SDK 调用失败", func() {
//     patches := ApplyMethodFunc(...)
//     defer patches.Reset()  // 禁止：defer 是函数级别的，不是 Convey 块级别的
// })

// ❌ 错误：不应打桩 util 层的普通方法
// patches = ApplyFunc(util.NewClient, ...)        // 禁止
// patches = ApplyFunc(util.Upload, ...)           // 禁止
// patches = ApplyFunc(util.GetBucketType, ...)    // 禁止
// patches = ApplyFunc(util.FormatUrl, ...)        // 禁止

// ✅ 例外：util 层中内部直接发起 HTTP 请求的方法需要打桩（如 util.CamAuth）
// util.CamAuth 内部使用 http.Client 直接发起 HTTP 请求，不经过 cos go SDK
Convey("CamAuth 打桩示例", func() {
    patches = ApplyFunc(util.CamAuth, func(roleName string) (util.Data, error) {
        return util.Data{TmpSecretId: "mock-id", TmpSecretKey: "mock-key", Token: "mock-token"}, nil
    })
    // ...
})
```

## 打桩覆盖范围要求

**只打桩 cos go SDK 的方法**，不得产生真实的外部请求：

| 调用类型 | 是否打桩 | 说明 |
|---|---|---|
| `cos.ObjectService` / `cos.BucketService` / `cos.ServiceService` / `cos.CIService` 的所有方法 | ✅ 必须打桩 | 会产生真实 HTTP 请求 |
| `util.*` 的大多数方法（`util.NewClient`、`util.Upload`、`util.GetBucketType`、`util.FormatUrl` 等） | ❌ 不打桩 | 让其正常执行，通过打桩内部 SDK 方法屏蔽网络请求 |
| `util.CamAuth` 等 util 层中**内部直接发起 HTTP 请求**的方法 | ✅ 需要打桩 | 不经过 cos go SDK，直接使用 `http.Client` 发起请求 |

## 测试配置初始化

**单测禁止依赖真实的 `~/.cos.yaml` 配置文件**，必须在测试开始时创建临时配置文件，测试结束后删除。

### 临时配置文件内容格式

```yaml
cos:
  base:
    secretid: "test-secret-id"
    secretkey: "test-secret-key"
    sessiontoken: ""
    protocol: "https"
  buckets:
    - name: "test-bucket-1234567890"
      alias: "test-alias"
      region: "ap-guangzhou"
      endpoint: "cos.ap-guangzhou.myqcloud.com"
      ofs: false
      customized: false
```

### 测试配置文件管理规范

```go
const testConfigPath = "/tmp/coscli-test.yaml"

// setupTestConfig：创建临时测试配置文件，并加载到全局 config
func setupTestConfig() {
    content := `cos:
  base:
    secretid: "test-secret-id"
    secretkey: "test-secret-key"
    sessiontoken: ""
    protocol: "https"
  buckets:
    - name: "test-bucket-1234567890"
      alias: "test-alias"
      region: "ap-guangzhou"
      endpoint: "cos.ap-guangzhou.myqcloud.com"
      ofs: false
      customized: false
`
    if err := os.WriteFile(testConfigPath, []byte(content), 0644); err != nil {
        logger.Errorln("创建测试配置文件失败:", err)
        return
    }
    // 加载临时配置文件
    viper.SetConfigFile(testConfigPath)
    if err := viper.ReadInConfig(); err != nil {
        logger.Errorln("读取测试配置文件失败:", err)
        return
    }
    if err := viper.UnmarshalKey("cos", &config); err != nil {
        logger.Errorln("解析测试配置文件失败:", err)
    }
}

// teardownTestConfig：删除临时测试配置文件
func teardownTestConfig() {
    if err := os.Remove(testConfigPath); err != nil && !os.IsNotExist(err) {
        logger.Errorln("删除测试配置文件失败:", err)
    }
}
```

### 在测试函数中使用

```go
func TestXxxCmd(t *testing.T) {
    setupTestConfig()           // 创建临时配置文件
    defer teardownTestConfig()  // 测试结束后删除

    Convey("Test coscli xxx", t, func() {
        // ...
    })
}
```

**注意**：`testconfig_test.go` 中的 `init()` 函数**不应调用** `setupTestConfig()`，避免与每个 `TestXxxCmd` 函数中的调用产生冗余。每个测试函数自己管理配置文件的生命周期。

`setUp` / `tearDown` 会产生真实的 COS API 调用（创建/删除桶），**单测中禁止使用**，仅在需要真实环境验证的集成测试中使用。

## 断言规范

```go
So(e, ShouldBeNil)     // 期望无错误
So(e, ShouldBeError)   // 期望有错误（不关心具体错误内容）
So(result, ShouldEqual, expected)
So(result, ShouldNotBeNil)
```


