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
- **任何向桶传递 `versionId` 的接口**：OFS 桶不接受 `versionId`，必须按桶类型决定是否携带（详见下文「OFS 桶与 versionId 处理」）

简单操作（`mb`、`rb`、`signurl` 等）若不涉及 `versionId` 通常不需要判断桶类型。

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

## OFS 桶与 versionId 处理（强制规则）

**OFS 桶不接受 `versionId`**，携带会导致请求异常。因此**任何向桶传递 `versionId` 的接口**在发起请求前都必须按桶类型判断是否携带，不能直接把用户传入的 `versionId` 透传给 SDK。

### 统一辅助函数 needCarryVersionId

```go
// util/copy.go
// needCarryVersionId 判断向指定类型的桶发起请求时是否应携带 versionId。
// 规则：OFS 桶不接受 versionId；且仅当用户显式指定了 versionId 时才携带。
func needCarryVersionId(bucketType, versionId string) bool {
    return bucketType != BucketTypeOfs && versionId != ""
}
```

**所有向桶传递 `versionId` 的判定都必须走这个函数**，禁止各处自行写 `if versionId != ""`。

### 标准使用模式

```go
// HEAD / IsExist 等可变参数接口：仅在需要时才把 versionId 作为可变参数传入
if needCarryVersionId(bucketType, versionId) {
    resp, err = c.Object.Head(context.Background(), objectKey, opt, versionId)
} else {
    resp, err = c.Object.Head(context.Background(), objectKey, opt)
}

// Options 结构体接口：仅在需要时才给 opt.VersionId 赋值
opt := &cos.ObjectDeleteOptions{ /* ... */ }
if needCarryVersionId(fo.BucketType, fo.Operation.VersionId) {
    opt.VersionId = fo.Operation.VersionId
}
```

### 已覆盖的链路（新增此类接口时须对照）

| 链路 | 接口 | 桶类型来源 |
|---|---|---|
| copy | `MultiCopy` / `GetHead` / `CheckCosObjectExist` | 源桶按源类型、目标桶按目标类型 |
| download | `Download` / `CheckCosObjectExist` | 源桶类型 |
| stat | `Object.Head` | `StatObject` 的 `bucketType` 入参 |
| rm | `CheckCosObjectExist` / `Object.Delete` | `fo.BucketType` |

### 开发新接口时的检查清单

1. 接口是否会向桶传递 `versionId`？是则必须判断桶类型。
2. 在 cmd 层 `NewClient` 后调用 `GetBucketType` 获取桶类型；util 层批量操作从 `fo.BucketType` 取。
3. 用 `needCarryVersionId` 统一判定，OFS 桶一律不携带 `versionId`。
4. 若 util 函数需要桶类型，给函数签名新增 `bucketType string` 参数（如 `StatObject`），由 cmd 层传入。
5. 补充 OFS/COS 两个断言子测试：直接捕获底层请求的 `id...` 可变参数或 `opt.VersionId`，验证 OFS 不携带、COS 携带。

---

## 注意事项

1. `GetBucketType` 内部会发送 HEAD Bucket 网络请求（默认模式），会消耗一次 API 调用
2. 若命令行传入了非法的 `--bucket-type` 值，会调用 `logger.Fatalln` 直接退出
3. 批量操作中，桶类型通过 `fo.BucketType` 传递给 util 层，不需要在 util 层再次判断
4. **OFS 桶不接受 `versionId`**：任何传 `versionId` 的接口都须用 `needCarryVersionId` 判定是否携带
