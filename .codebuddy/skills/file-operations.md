# Skill：FileOperations 批量操作配置

## 概述

`FileOperations` 是批量文件操作（cp/sync/rm）的核心上下文结构体，承载所有操作参数、监控器、日志器和配置引用。

---

## 完整字段说明

```go
type FileOperations struct {
    Operation     Operation            // 操作参数（见下方）
    Monitor       *FileProcessMonitor  // 进度监控，批量操作必须初始化
    ErrOutput     *ErrOutput           // 错误输出文件，有 fail-output 时必须初始化
    ProcessLogger *ProcessLogger       // 进程日志，有 process-log 时必须初始化
    Config        *Config              // 配置引用，必须
    Param         *Param               // 参数引用，必须
    SnapshotDb    *leveldb.DB          // 快照数据库，有 snapshot-path 时由 InitSnapshotDb 初始化
    CpType        CpType               // 操作类型：CpTypeUpload/CpTypeDownload/CpTypeCopy
    Command       string               // 命令名称，使用 const.go 中的常量
    DeleteCount   int                  // 删除计数（rm 命令使用）
    BucketType    string               // 桶类型，由 GetBucketType 填充
    OutPutDirName string               // 输出目录名，time.Now().Format("20060102_150405")
}
```

---

## Operation 常用字段

```go
type Operation struct {
    Recursive         bool              // -r 递归操作
    Filters           []FilterOptionType // --include/--exclude 过滤规则
    StorageClass      string            // --storage-class 存储类型
    RateLimiting      float32           // --rate-limiting 速率限制(MB/s)
    PartSize          int64             // --part-size 分块大小(MB)，默认 32
    ThreadNum         int               // --thread-num 线程数
    Routines          int               // --routines 并发文件数，默认 3
    FailOutput        bool              // --fail-output 是否输出错误文件
    FailOutputPath    string            // --fail-output-path 错误文件目录
    ProcessLog        bool              // --process-log 是否记录进程日志
    ProcessLogPath    string            // --process-log-path 进程日志目录
    RetryNum          int               // --retry-num 限速重试次数(0-100)
    ErrRetryNum       int               // --err-retry-num 错误重试次数(0-100)
    ErrRetryInterval  int               // --err-retry-interval 重试间隔(0-10秒)
    CheckPoint        bool              // --check-point 断点续传，默认 true
    DisableCrc64      bool              // --disable-crc64 禁用 CRC64 校验
    DisableLongLinks  bool              // --disable-long-links 禁用长连接
    LongLinksNums     int               // --long-links-nums 长连接数量
    Delete            bool              // --delete 同步删除目标多余文件
    Force             bool              // --force 强制执行不提示
    Update            bool              // --update 仅更新更新的文件
    IgnoreExisting    bool              // --ignore-existing 忽略已存在文件
    Move              bool              // --move COS 间移动（拷贝后删除源）
    // ... 其他字段见 util/types.go
}
```

---

## 标准构造模式

```go
fo := &util.FileOperations{
    Operation: util.Operation{
        Recursive:        recursive,
        Filters:          filters,
        StorageClass:     storageClass,
        RateLimiting:     rateLimiting,
        PartSize:         partSize,
        ThreadNum:        threadNum,
        Routines:         routines,
        FailOutput:       failOutput,
        FailOutputPath:   failOutputPath,
        ProcessLog:       processLog,
        ProcessLogPath:   processLogPath,
        RetryNum:         retryNum,
        ErrRetryNum:      errRetryNum,
        ErrRetryInterval: errRetryInterval,
        CheckPoint:       checkPoint,
        // ... 按需添加
    },
    Monitor:       &util.FileProcessMonitor{},
    Config:        &config,
    Param:         &param,
    ErrOutput:     &util.ErrOutput{},
    ProcessLogger: &util.ProcessLogger{},
    CpType:        getCommandType(srcUrl, destUrl),  // 上传/下载/拷贝
    Command:       util.CommandCP,
    OutPutDirName: time.Now().Format("20060102_150405"),
}
```

---

## 快照数据库初始化

```go
// 在构造 fo 之后，执行操作之前
err = util.InitSnapshotDb(srcUrl, destUrl, fo)
if err != nil {
    return err
}
```

---

## 操作类型判断

```go
// getCommandType 根据 src/dest URL 类型判断操作方向
// 定义在 cmd/cp.go 或 cmd/sync.go 中
func getCommandType(srcUrl, destUrl util.StorageUrl) util.CpType {
    if srcUrl.IsFileUrl() && destUrl.IsCosUrl() {
        return util.CpTypeUpload
    } else if srcUrl.IsCosUrl() && destUrl.IsFileUrl() {
        return util.CpTypeDownload
    }
    return util.CpTypeCopy
}
```

---

## 过滤规则解析

```go
include, _ := cmd.Flags().GetString("include")
exclude, _ := cmd.Flags().GetString("exclude")

// GetFilter 返回 (hasFilter bool, filters []FilterOptionType)
_, filters := util.GetFilter(include, exclude)

// 注意：有过滤规则时必须开启 -r，否则报错
if len(filters) > 0 && !recursive {
    return fmt.Errorf("--include or --exclude can only use with --recursive option")
}
```

---

## 操作结束清理

批量操作结束后必须执行：

```go
util.CloseErrorOutputFile(fo)    // 关闭错误输出文件
util.CloseProcessLoggerFile(fo)  // 关闭进程日志文件
endT := time.Now().UnixNano() / 1000 / 1000
util.PrintCostTime(startT, endT) // 输出耗时

// 根据错误数量决定日志级别和退出码
if fo.Monitor.ErrNum > 0 {
    logger.Warningf("...")
    os.Exit(2)  // 部分失败退出码
} else {
    logger.Infof("...")
}
```
