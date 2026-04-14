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

## 测试函数结构

**所有测试用例中涉及外部服务调用的部分（COS API、网络请求等）都必须使用打桩方式，禁止在单测中产生真实的外部服务调用。**

```go
func TestXxxCmd(t *testing.T) {
    Convey("Test coscli xxx", t, func() {
        Convey("正常场景描述", func() {
            // 打桩所有外部服务调用
            patches := ApplyFunc(util.NewClient, func(*util.Config, *util.Param, string, ...*util.FileOperations) (*cos.Client, error) {
                return &cos.Client{}, nil
            })
            defer patches.Reset()
            patches.ApplyFunc(util.XxxFunc, func(...) error {
                return nil  // 模拟成功
            })

            clearCmd()  // 每个子用例必须先 clearCmd()
            cmd := rootCmd
            args := []string{"xxx", "cos://bucket/key", "--flag", "value"}
            cmd.SetArgs(args)
            e := cmd.Execute()
            So(e, ShouldBeNil)
        })
        Convey("错误场景描述", func() {
            // 打桩模拟外部服务失败
            patches := ApplyFunc(util.NewClient, func(*util.Config, *util.Param, string, ...*util.FileOperations) (*cos.Client, error) {
                return nil, fmt.Errorf("mock NewClient error")
            })
            defer patches.Reset()

            clearCmd()
            cmd := rootCmd
            args := []string{"xxx", "cos://bucket/key"}
            cmd.SetArgs(args)
            e := cmd.Execute()
            So(e, ShouldBeError)
        })
        Convey("参数校验错误（无需打桩）", func() {
            // 纯参数校验不涉及外部调用，无需打桩
            clearCmd()
            cmd := rootCmd
            args := []string{"xxx", "invalid-path"}
            cmd.SetArgs(args)
            e := cmd.Execute()
            So(e, ShouldBeError)
        })
    })
}
```

## clearCmd() 规范

**每个 Convey 子用例开头必须调用 `clearCmd()`**，重置所有 Flag 到默认值：

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

```go
// 打桩普通函数
patches := ApplyFunc(util.NewClient, func(config *util.Config, param *util.Param, bucketName string) (*cos.Client, error) {
    return nil, fmt.Errorf("mock error")
})
defer patches.Reset()

// 打桩方法（需要 reflect.TypeOf）
var c *cos.BucketService
patches := ApplyMethodFunc(reflect.TypeOf(c), "Head", func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
    return nil, fmt.Errorf("test Head error")
})
defer patches.Reset()

// 多个打桩叠加（使用同一个 patches 对象）
patches := ApplyFunc(util.FormatDownloadPath, func(...) error {
    return fmt.Errorf("test error")
})
defer patches.Reset()
var h http.Header
patches.ApplyMethodFunc(h, "Get", func(key string) string {
    if key == "X-Cos-Bucket-Arch" {
        return "OFS"
    }
    return ""
})
```

## 打桩覆盖范围要求

以下类型的调用**必须**打桩，不得产生真实的外部请求：

| 调用类型 | 必须打桩的函数 |
|---|---|
| Client 创建 | `util.NewClient`、`util.CreateClient` |
| COS 对象操作 | `util.Upload`、`util.Download`、`util.CosCopy`、`util.DeleteObjects` 等 |
| 路径格式化 | `util.FormatUploadPath`、`util.FormatDownloadPath`、`util.FormatCopyPath` |
| 桶类型获取 | `util.GetBucketType`（会发 HEAD Bucket 请求） |
| SDK 方法 | `cos.BucketService.Head`、`cos.ObjectService.Head` 等所有 SDK 方法 |
| 路径检查 | `util.CheckPath`（可能访问本地文件系统或 COS） |

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

`setUp` / `tearDown` 会产生真实的 COS API 调用（创建/删除桶），**单测中禁止使用**，仅在需要真实环境验证的集成测试中使用。

## 断言规范

```go
So(e, ShouldBeNil)     // 期望无错误
So(e, ShouldBeError)   // 期望有错误（不关心具体错误内容）
So(result, ShouldEqual, expected)
So(result, ShouldNotBeNil)
```


