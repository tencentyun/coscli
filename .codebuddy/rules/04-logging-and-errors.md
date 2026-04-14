# 规则：日志与错误输出

## 日志框架

使用 `github.com/sirupsen/logrus`，以别名 `logger` 导入：

```go
import logger "github.com/sirupsen/logrus"
```

## 日志级别使用规范

| 级别 | 函数 | 使用场景 |
|---|---|---|
| Info | `logger.Infof / logger.Infoln` | 操作开始、操作成功、进度结果 |
| Warning | `logger.Warningf / logger.Warningln` | 操作完成但有部分错误（不中断） |
| Error | `logger.Errorf / logger.Errorln` | 测试辅助代码中的错误（不退出） |
| Fatal | `logger.Fatalln` | 配置/参数不可恢复的错误（会 os.Exit(1)） |

## 标准日志输出模式

```go
// 操作开始
logger.Infof("Upload %s to %s start", srcPath, destPath)
logger.Infof("Download %s to %s start", srcPath, destPath)
logger.Infof("Copy %s to %s start", srcPath, destPath)

// 操作成功（含 Monitor 统计信息）
logger.Infof("%s %s to %s %s", operate, srcPath, destPath, fo.Monitor.GetFinishInfo())

// 操作有部分错误（ErrNum > 0 时）
if fo.Monitor.ErrNum > 0 {
    logger.Warningf("%s %s to %s %s", operate, srcPath, destPath, fo.Monitor.GetFinishInfo())
    os.Exit(2)  // 退出码 2 表示部分失败
} else {
    logger.Infof("%s %s to %s %s", operate, srcPath, destPath, fo.Monitor.GetFinishInfo())
}

// 单个资源操作成功
logger.Infof("Create a new bucket! name: %s\n", bucketIDName)
logger.Infof("Add successfully! name: %s, endpoint: %s, alias: %s, ofs: %t, customized: %t\n", ...)
logger.Infoln("Modify successfully!")

// 不可恢复错误（仅用于配置解析阶段）
logger.Fatalln("bucket type can only be either COS or OFS ")
logger.Fatalln("missing parameter SecretID")
```

## 错误返回规范

**原则：业务错误通过 `return fmt.Errorf(...)` 返回给 cobra，不在 util 层直接打印或退出。**

```go
// ✅ 正确：util 层返回错误
func SomeUtilFunc(...) error {
    if err != nil {
        return err  // 或 return fmt.Errorf("context: %v", err)
    }
    return nil
}

// ✅ 正确：cmd 层处理错误
RunE: func(cmd *cobra.Command, args []string) error {
    err := util.SomeUtilFunc(...)
    if err != nil {
        return err  // cobra 会打印错误信息
    }
    return nil
}

// ❌ 错误：util 层直接打印并退出
func SomeUtilFunc(...) {
    if err != nil {
        logger.Fatalln(err)  // 禁止，除非是配置解析阶段
    }
}
```

## 日志初始化

日志由 `root.go` 的 `initConfig()` 自动初始化，无需在命令中手动调用：

```go
// root.go initConfig() 中
clilog.InitLoggerWithDir(logPath, disableLog)
```

日志文件默认输出到可执行文件同目录的 `coscli.log`，同时输出到 stdout。
日志按天轮转，保留 7 天（168 小时）。

## 耗时统计

批量操作结束后输出耗时：

```go
startT := time.Now().UnixNano() / 1000 / 1000
// ... 执行操作 ...
endT := time.Now().UnixNano() / 1000 / 1000
util.PrintCostTime(startT, endT)
```

## 错误输出文件（fail-output）

批量操作失败的文件记录到错误输出文件，操作结束后必须关闭：

```go
util.CloseErrorOutputFile(fo)
util.CloseProcessLoggerFile(fo)
```
