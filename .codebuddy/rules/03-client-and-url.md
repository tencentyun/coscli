# 规则：Client 实例化与 URL 生成

## NewClient vs CreateClient 选择规则

| 场景 | 使用函数 | 原因 |
|---|---|---|
| 操作**已在配置文件注册**的桶（含批量传输） | `util.NewClient(&config, &param, bucketName, fo)` | 自动查配置、生成正确 URL、支持长连接池、支持 customized 域名 |
| 操作**已在配置文件注册**的桶（简单操作） | `util.NewClient(&config, &param, bucketName)` | 不传 `fo`，使用默认连接池 |
| 列出所有桶（Service 请求） | `util.NewClient(&config, &param, "")` | bucketName 传空字符串，生成 ServiceURL |
| 操作**未注册的桶**（如 mb/rb 命令目标） | `util.CreateClient(&config, &param, bucketIDName)` | 直接用 `-e endpoint` 参数构造 URL，不查配置 |

> **混用会导致 customized 域名逻辑失效**，这是最常见的错误。

## NewClient 签名

```go
func NewClient(config *Config, param *Param, bucketName string, options ...*FileOperations) (*cos.Client, error)
```

- `bucketName` 传入的是**别名（alias）或桶名**，函数内部调用 `FindBucket` 查找配置
- `options` 为可变参数，传入 `fo` 时启用长连接池（连接数 = `fo.Operation.Routines`）
- 内部调用 `GenURL` 生成 URL，`GenURL` 中实现了 customized 优先级逻辑

## CreateClient 签名

```go
func CreateClient(config *Config, param *Param, bucketIDName string) (*cos.Client, error)
```

- `bucketIDName` 是完整桶名（含 appid，如 `examplebucket-1234567890`）
- 直接使用 `param.Endpoint` 构造 URL，不查配置文件中的桶列表
- 不支持 customized 域名（`CreateURL` 第四参数固定传 `false`）

## URL 生成链路

```
NewClient
  └─ GenURL(config, param, bucketName)
       ├─ FindBucket(config, bucketName)  // 查配置文件
       ├─ 优先级合并 endpoint/protocol
       ├─ customized := param.Customized || bucket.Customized
       └─ CreateURL(idName, protocol, endpoint, customized)
            ├─ GenBucketURL → "https://bucket.endpoint" 或 "https://endpoint"
            ├─ GenServiceURL → "https://endpoint"
            └─ GenCiURL → "https://bucket.endpoint"

CreateClient
  └─ CreateURL(bucketIDName, protocol, param.Endpoint, false)
```

## customized 域名的效果

```go
// customized = false（默认）：标准 COS 域名
// "https://examplebucket-1234567890.cos.ap-guangzhou.myqcloud.com"

// customized = true：自定义域名（endpoint 作为完整域名）
// "https://my-custom-domain.com"
```

## COS URL 解析规范

```go
// 解析 cos:// 路径（推荐方式）
cosUrl, err := util.FormatUrl(args[0])
if err != nil {
    return fmt.Errorf("cos url format error:%v", err)
}
if !cosUrl.IsCosUrl() {
    return fmt.Errorf("cospath needs to contain cos://")
}
bucketName := cosUrl.(*util.CosUrl).Bucket  // 桶名（可能是 alias）
objectKey  := cosUrl.(*util.CosUrl).Object  // 对象路径

// 仅需桶名时的简化方式（mb/rb 等命令）
bucketIDName, cosPath := util.ParsePath(args[0])
// args[0] = "cos://examplebucket-1234567890" → bucketIDName="examplebucket-1234567890", cosPath=""
```

## 错误处理规范

```go
c, err := util.NewClient(&config, &param, bucketName)
if err != nil {
    return err  // 直接向上返回，不包装
}

// 业务错误需要上下文时才包装
cosUrl, err := util.FormatUrl(args[0])
if err != nil {
    return fmt.Errorf("cos url format error:%v", err)
}
```

## 长连接配置

批量操作（cp/sync）传入 `fo` 时，连接池大小由以下逻辑决定：

```go
// 见 util/client.go NewClient
if !fo.Operation.DisableLongLinks {
    longLinksNums := fo.Operation.LongLinksNums
    if longLinksNums == 0 {
        longLinksNums = fo.Operation.Routines  // 默认等于并发数
    }
    // MaxIdleConnsPerHost = MaxIdleConns = longLinksNums
}
```

## 重试配置

所有 Client 默认配置：
- 重试次数：`client.Conf.RetryOpt.Count = 10`
- 重试间隔：`client.Conf.RetryOpt.Interval = 1`（秒）
- 备用域名切换：`CloseAutoSwitchHost == "false"` 时开启（注意是字符串 "false"）
