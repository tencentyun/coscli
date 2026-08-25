---
type: always
---

# 规则：versionId 与桶类型（OFS 不接受 versionId）

## 核心约束

**OFS 桶不接受 `versionId`**，携带会导致请求异常。因此**任何向桶传递 `versionId` 的接口**，在发起请求前都必须按桶类型判断是否携带，禁止把用户传入的 `versionId` 直接透传给 SDK。

## 统一判定函数

所有 `versionId` 携带判定都必须走 `util/copy.go` 的 `needCarryVersionId`，禁止各处自行写 `if versionId != ""`：

```go
func needCarryVersionId(bucketType, versionId string) bool {
    return bucketType != BucketTypeOfs && versionId != ""
}
```

## 使用模式

```go
// 可变参数接口（HEAD / IsExist 等）
if needCarryVersionId(bucketType, versionId) {
    resp, err = c.Object.Head(ctx, objectKey, opt, versionId)
} else {
    resp, err = c.Object.Head(ctx, objectKey, opt)
}

// Options 结构体接口（Delete / Copy 等）
if needCarryVersionId(fo.BucketType, fo.Operation.VersionId) {
    opt.VersionId = fo.Operation.VersionId
}
```

## 桶类型来源

- cmd 层：`NewClient` 之后调用 `util.GetBucketType(c, &param, &config, bucketName)` 获取，并传入 util 函数。
- util 层批量操作：从 `fo.BucketType` 取（由 cmd 层预先写入）。
- 若 util 函数需要桶类型，给其签名新增 `bucketType string` 参数（如 `StatObject`）。

## 已覆盖链路（新增同类接口须对照）

| 链路 | 接口 |
|---|---|
| copy | `MultiCopy` / `GetHead` / `CheckCosObjectExist` |
| download | `Download` / `CheckCosObjectExist` |
| stat | `Object.Head`（`StatObject`） |
| rm | `CheckCosObjectExist` / `Object.Delete` |

## 测试要求

新增或修改此类接口时，必须补充 OFS / COS 两个断言子测试：直接捕获底层请求的 `id...` 可变参数或 `opt.VersionId`，验证 **OFS 桶不携带、COS 桶携带** `versionId`。

> 详细说明见 skill：`coscli-develop-skills/references/bucket-type-detection.md`「OFS 桶与 versionId 处理」。
