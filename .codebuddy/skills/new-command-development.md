# Skill：新命令开发完整流程

## 概述

本 Skill 描述在 coscli 中开发一个新命令的完整步骤，以一个假设的 `stat` 命令（获取对象元数据）为例。

---

## Step 1：确定命令类型

根据命令特征选择模板：

| 类型 | 特征 | 参考命令 |
|---|---|---|
| **简单操作** | 单个对象，无批量，无 FileOperations | `ls`, `signurl`, `mb`, `rb` |
| **批量操作** | 多文件，需 FileOperations + Monitor | `cp`, `sync`, `rm` |
| **配置操作** | 修改配置文件 | `config add`, `config set` |

---

## Step 2：创建命令文件

在 `cmd/` 下创建 `<name>.go`，按固定结构编写：

### 简单操作模板（以 stat 为例）

```go
package cmd

import (
    "coscli/util"
    "fmt"

    logger "github.com/sirupsen/logrus"
    "github.com/spf13/cobra"
)

var statCmd = &cobra.Command{
    Use:   "stat",
    Short: "Get object metadata",
    Long: `Get object metadata

Format:
  ./coscli stat cos://<bucket-name>[/object-key] [flags]

Example:
  ./coscli stat cos://examplebucket/test/example.txt`,
    Args: cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        // ① 读取 Flags
        versionId, _ := cmd.Flags().GetString("version-id")

        // ② 参数校验
        cosUrl, err := util.FormatUrl(args[0])
        if err != nil {
            return fmt.Errorf("cos url format error:%v", err)
        }
        if !cosUrl.IsCosUrl() {
            return fmt.Errorf("cospath needs to contain cos://")
        }

        // ③ 提取桶名和对象路径
        bucketName := cosUrl.(*util.CosUrl).Bucket
        objectKey  := cosUrl.(*util.CosUrl).Object

        // ④ 实例化 Client（已注册桶用 NewClient）
        c, err := util.NewClient(&config, &param, bucketName)
        if err != nil {
            return err
        }

        // ⑤ 调用 util 层业务函数
        err = util.StatObject(c, objectKey, versionId)
        if err != nil {
            return err
        }

        // ⑥ 成功日志
        logger.Infof("stat %s success", args[0])
        return nil
    },
}

func init() {
    rootCmd.AddCommand(statCmd)
    statCmd.Flags().String("version-id", "", "Specify the version ID of the object")
}
```

### 批量操作模板（FileOperations）

```go
fo := &util.FileOperations{
    Operation: util.Operation{
        Recursive:      recursive,
        Filters:        filters,
        FailOutput:     failOutput,
        FailOutputPath: failOutputPath,
        ProcessLog:     processLog,
        ProcessLogPath: processLogPath,
        Routines:       routines,
        // ... 其他操作参数
    },
    Monitor:       &util.FileProcessMonitor{},
    Config:        &config,
    Param:         &param,
    ErrOutput:     &util.ErrOutput{},
    ProcessLogger: &util.ProcessLogger{},
    Command:       util.CommandCP,  // 使用 const.go 中的常量
    OutPutDirName: time.Now().Format("20060102_150405"),
}

bucketName := cosUrl.(*util.CosUrl).Bucket
c, err := util.NewClient(fo.Config, fo.Param, bucketName, fo)  // 传 fo 启用长连接
if err != nil {
    return err
}

// 获取桶类型（需区分 COS/OFS 时）
fo.BucketType, err = util.GetBucketType(c, fo.Param, fo.Config, bucketName)
if err != nil {
    return err
}

// 操作结束后
util.CloseErrorOutputFile(fo)
util.CloseProcessLoggerFile(fo)
endT := time.Now().UnixNano() / 1000 / 1000
util.PrintCostTime(startT, endT)

if fo.Monitor.ErrNum > 0 {
    logger.Warningf("%s %s to %s %s", operate, srcPath, destPath, fo.Monitor.GetFinishInfo())
    os.Exit(2)
} else {
    logger.Infof("%s %s to %s %s", operate, srcPath, destPath, fo.Monitor.GetFinishInfo())
}
```

---

## Step 3：在 util/ 层实现业务逻辑

```go
// util/stat.go
package util

import (
    "context"
    "fmt"
    "github.com/tencentyun/cos-go-sdk-v5"
    logger "github.com/sirupsen/logrus"
)

func StatObject(c *cos.Client, objectKey string, versionId string) error {
    opt := &cos.ObjectHeadOptions{}
    if versionId != "" {
        opt.XCosSSECustomerAglo = versionId  // 示例
    }

    resp, err := c.Object.Head(context.Background(), objectKey, opt)
    if err != nil {
        return err  // 直接返回，不打印
    }

    // 输出结果
    logger.Infof("Content-Type: %s", resp.Header.Get("Content-Type"))
    logger.Infof("Content-Length: %s", resp.Header.Get("Content-Length"))
    return nil
}
```

---

## Step 4：添加常量（如需要）

在 `util/const.go` 中添加新命令相关常量：

```go
const (
    CommandCP      = "cp"
    CommandSync    = "sync"
    CommandStat    = "stat"  // 新增
    // ...
)
```

---

## Step 5：编写测试文件

**核心原则：**
1. **只对直接调用 cos go SDK 的方法打桩，禁止在单测中产生真实的外部服务调用。util 层的所有方法（包括 `util.NewClient`、`util.Upload`、`util.GetBucketType` 等）不需要打桩，让它们正常执行。**
2. **禁止依赖真实的 `~/.cos.yaml`，必须创建临时测试配置文件，测试结束后删除。**
3. **单测覆盖率必须达到 95% 以上，必须覆盖命令的所有执行分支。**

### 覆盖率达标清单

每个命令的测试用例必须覆盖以下所有分支：

| 分支类型 | 测试用例 | 是否必须 |
|---|---|---|
| 参数数量不足 | 不传参数或参数不够 | ✅ |
| URL 格式错误 | 传入非 `cos://` 路径 | ✅ |
| SDK 调用失败 | 打桩 cos SDK 方法返回 error | ✅ |
| 成功路径 | 打桩 cos SDK 方法返回成功 | ✅ |
| 各 Flag 组合 | 覆盖所有重要 Flag | ✅ |

### 覆盖率检查命令

```bash
# 运行单个命令的测试并查看覆盖率
go test -v -gcflags="all=-l" -coverprofile=coverage.out ./cmd/ -run TestStatCmd
go tool cover -func=coverage.out | grep -E "(stat|total)"

# 查看 HTML 覆盖报告
go tool cover -html=coverage.out -o coverage.html
```

创建 `cmd/stat_test.go`：

```go
package cmd

import (
    "context"
    "fmt"
    "os"
    "reflect"
    "testing"

    . "github.com/agiledragon/gomonkey/v2"
    . "github.com/smartystreets/goconvey/convey"
    "coscli/util"
    "github.com/spf13/viper"
    "github.com/tencentyun/cos-go-sdk-v5"
)

const statTestConfigPath = "/tmp/coscli-stat-test.yaml"

func setupStatTestConfig() {
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
    os.WriteFile(statTestConfigPath, []byte(content), 0644)
    viper.SetConfigFile(statTestConfigPath)
    viper.ReadInConfig()
    viper.UnmarshalKey("cos", &config)
}

func teardownStatTestConfig() {
    os.Remove(statTestConfigPath)
}

func TestStatCmd(t *testing.T) {
    setupStatTestConfig()           // 创建临时配置文件
    defer teardownStatTestConfig()  // 测试结束后删除

    Convey("Test coscli stat", t, func() {
        Convey("路径不含 cos://（纯参数校验，无需打桩）", func() {
            clearCmd()
            cmd := rootCmd
            args := []string{"stat", "invalid-path"}
            cmd.SetArgs(args)
            e := cmd.Execute()
            So(e, ShouldBeError)
        })
        Convey("SDK 调用失败", func() {
            // 只打桩 cos SDK 方法
            var obj *cos.ObjectService
            patches := ApplyMethodFunc(reflect.TypeOf(obj), "Head",
                func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
                    return nil, fmt.Errorf("mock sdk error")
                })
            defer patches.Reset()

            clearCmd()
            cmd := rootCmd
            args := []string{"stat", "cos://test-alias/test-object"}
            cmd.SetArgs(args)
            e := cmd.Execute()
            So(e, ShouldBeError)
        })
        Convey("成功获取对象元数据", func() {
            // 只打桩 cos SDK 方法
            var obj *cos.ObjectService
            patches := ApplyMethodFunc(reflect.TypeOf(obj), "Head",
                func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
                    return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
                })
            defer patches.Reset()

            clearCmd()
            cmd := rootCmd
            args := []string{"stat", "cos://test-alias/test-object"}
            cmd.SetArgs(args)
            e := cmd.Execute()
            So(e, ShouldBeNil)
        })
        Convey("带 --version-id 参数的成功路径", func() {
            var obj *cos.ObjectService
            patches := ApplyMethodFunc(reflect.TypeOf(obj), "Head",
                func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
                    return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
                })
            defer patches.Reset()

            clearCmd()
            cmd := rootCmd
            args := []string{"stat", "cos://test-alias/test-object", "--version-id", "v1"}
            cmd.SetArgs(args)
            e := cmd.Execute()
            So(e, ShouldBeNil)
        })
    })
}
```

### 打桩边界原则

**只对直接调用 cos go SDK 的方法打桩**，其余所有方法正常执行：

| 类型 | 是否打桩 | 说明 |
|---|---|---|
| `cos.ObjectService` 的所有方法 | ✅ 必须打桩 | 会产生真实 HTTP 请求 |
| `cos.BucketService` 的所有方法 | ✅ 必须打桩 | 会产生真实 HTTP 请求 |
| `cos.ServiceService` 的所有方法 | ✅ 必须打桩 | 会产生真实 HTTP 请求 |
| `util.NewClient`、`util.Upload`、`util.GetBucketType` 等 | ❌ 不打桩 | 让其正常执行，通过打桩内部 SDK 方法屏蔽网络请求 |
| `util.FormatUrl`、`util.GetFilter` 等纯逻辑函数 | ❌ 不打桩 | 纯本地逻辑，应正常执行以提升覆盖率 |
| `util.CamAuth` 等 util 层中**内部直接发起 HTTP 请求**的方法 | ✅ 需要打桩 | 不经过 cos go SDK，直接使用 `http.Client` 发起请求 |

---

## Step 6：验证 Long 描述格式

每个命令的 `Long` 字段必须包含 `Format:` 和 `Example:` 两个段落：

```go
Long: `命令功能一句话描述

Format:
  ./coscli <cmd> <必填参数> [可选参数] [flags]

Example:
  ./coscli <cmd> cos://examplebucket/key --flag value`,
```

---

## Step 7：全局参数检查

以下全局参数已在 `root.go` 中定义，**禁止**在子命令中重复声明：

| 参数 | 变量 | 说明 |
|---|---|---|
| `-c / --config-path` | `cfgFile` | 配置文件路径 |
| `-i / --secret-id` | `param.SecretID` | SecretID |
| `-k / --secret-key` | `param.SecretKey` | SecretKey |
| `--token` | `param.SessionToken` | SessionToken |
| `-e / --endpoint` | `param.Endpoint` | 自定义 endpoint |
| `--customized` | `param.Customized` | 使用自定义域名 |
| `-p / --protocol` | `param.Protocol` | 协议（http/https） |
| `--init-skip` | `initSkip` | 跳过配置初始化 |
| `--log-path` | `logPath` | 日志目录 |
| `--disable-log` | `disableLog` | 禁用日志 |
| `--close_auto_switch_host` | `param.CloseAutoSwitchHost` | 关闭备用域名切换 |
| `--bucket-type` | `param.BucketType` | 指定桶类型 COS/OFS |
