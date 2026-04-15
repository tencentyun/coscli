# Skill：桶类型判断与 OFS/COS 分支处理

## 概述

coscli 支持两种桶类型：
- **COS**：标准对象存储桶（默认）
- **OFS**：融合桶（Hadoop 兼容，部分操作行为不同）

---

## 何时需要判断桶类型

以下操作需要区分 COS/OFS：
- `ls`：OFS 用 `ListOfsObjects`，COS 用 `ListObjects`
- `cp`/`sync`：OFS 有特殊的目录处理逻辑
- `rm`：OFS 有特殊的删除逻辑

简单操作（`mb`、`rb`、`signurl`、`stat` 等）通常不需要判断桶类型。

---

## GetBucketType 函数

```go
// util/get_bucket_type.go
func GetBucketType(c *cos.Client, param *Param, config *Config, bucketName string) (string, error)
```

### 判断逻辑（优先级从高到低）

```
1. param.BucketType != ""
   → 直接使用命令行 --bucket-type 参数（必须是 "COS" 或 "OFS"，否则 Fatalln）

2. config.Base.DisableAutoFetchBucketType == "true"
   → 从配置文件中查找桶的 Ofs 字段决定类型（不发网络请求）

3. 默认
   → 发送 HEAD Bucket 请求，读取响应头 X-Cos-Bucket-Arch
     - 值为 "OFS" → OFS 桶
     - 其他 → COS 桶
```

---

## 标准使用模式

```go
// 在 NewClient 之后立即调用
c, err := util.NewClient(&config, &param, bucketName)
if err != nil {
    return err
}

bucketType, err := util.GetBucketType(c, &param, &config, bucketName)
if err != nil {
    return err
}

// 根据桶类型分支处理
if bucketType == util.BucketTypeOfs {
    // OFS 分支
    err = util.ListOfsObjects(c, cosUrl, limit, recursive, filters)
} else {
    // COS 分支（默认）
    err = util.ListObjects(c, cosUrl, limit, recursive, filters)
}
```

### 使用 FileOperations 时

```go
fo.BucketType, err = util.GetBucketType(c, fo.Param, fo.Config, bucketName)
if err != nil {
    return err
}
// fo.BucketType 会在 util 层的批量操作函数中使用
```

---

## 常量定义

```go
// util/const.go
const (
    BucketTypeCos = "COS"
    BucketTypeOfs = "OFS"
)
```

---

## 配置文件中禁用自动获取

当网络受限或需要提升性能时，可在配置文件中禁用自动获取：

```yaml
cos:
  base:
    disableautofetchbuckettype: "true"
  buckets:
    - name: "my-ofs-bucket-1234567890"
      ofs: true   # 手动标记为 OFS 桶
```

---

## 注意事项

1. `GetBucketType` 内部会发送 HEAD Bucket 网络请求（默认模式），会消耗一次 API 调用
2. 若命令行传入了非法的 `--bucket-type` 值，会调用 `logger.Fatalln` 直接退出
3. 批量操作中，桶类型通过 `fo.BucketType` 传递给 util 层，不需要在 util 层再次判断
