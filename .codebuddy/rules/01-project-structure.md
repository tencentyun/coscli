---
type: always
---

# 规则：项目结构与代码组织

## 目录职责

```
coscli/
├── main.go              # 程序入口，仅调用 cmd.Execute()
├── cmd/                 # 所有 cobra 命令定义，每个命令一个文件
│   ├── root.go          # 根命令、全局变量、initConfig()
│   ├── <command>.go     # 每个命令对应一个独立文件，文件名即命令名
│   └── *_test.go        # 集成测试文件，与命令文件同包
├── util/                # 业务逻辑层，不依赖 cobra
│   ├── types.go         # 所有结构体定义（Config/Param/FileOperations/Operation 等）
│   ├── const.go         # 所有常量定义
│   ├── client.go        # NewClient / CreateClient
│   ├── url.go           # URL 生成（GenURL / GenBucketURL / CreateURL 等）
│   ├── storage_url.go   # COS URL 解析（FormatUrl / ParsePath / CosUrl 等）
│   └── *.go             # 各业务功能实现
└── logger/
    └── logger.go        # 日志初始化（InitLoggerWithDir）
```

## 全局变量（定义在 cmd/root.go）

以下变量在所有命令文件中直接使用，**禁止**在子命令中重复声明：

```go
var cfgFile string       // -c 配置文件路径
var config util.Config   // 从配置文件反序列化的配置
var param util.Param     // 从命令行参数收集的运行时参数
var cmdCnt int           // 控制某些函数在一个命令中被调用的次数
```

## 命令文件固定结构

每个 `cmd/<name>.go` 文件必须按以下顺序组织：

```go
package cmd

// 1. import 块
import (...)

// 2. cobra.Command 变量定义
var xxxCmd = &cobra.Command{
    Use:   "xxx",
    Short: "一句话描述",
    Long:  `详细描述\n\nFormat:\n  ...\n\nExample:\n  ...`,
    Args:  cobra.ExactArgs(N),  // 或 MaximumNArgs / 自定义 func
    RunE:  func(cmd *cobra.Command, args []string) error { ... },
}

// 3. init() 函数：注册命令 + 声明 Flags
func init() {
    rootCmd.AddCommand(xxxCmd)  // 或 parentCmd.AddCommand(xxxCmd)
    xxxCmd.Flags().XxxP(...)
}

// 4. 业务逻辑函数（可选，复杂逻辑抽取为独立函数）
func doXxx(cmd *cobra.Command, args []string) error { ... }
```

## RunE 内部固定执行顺序

```
① 读取所有 Flags（cmd.Flags().GetXxx）
② 参数合法性校验（return fmt.Errorf(...)）
③ 解析 URL（util.FormatUrl / util.ParsePath）
④ 构造 FileOperations（批量操作时必须）
⑤ 实例化 Client（util.NewClient / util.CreateClient）
⑥ 获取桶类型（util.GetBucketType，需区分 COS/OFS 时）
⑦ 调用 util 层业务函数
⑧ 日志输出（logger.Infof / logger.Warningf）
```
