# API 快速参考

> 本文档为接口调用的速查手册，完整说明请参见 `SKILL.md`。

---

## 配置

- **Base URL**：`https://write.mcp.it.woa.com`
- **Token**：从环境变量 `AI_WRITE_API_TOKEN` 读取

## 通用请求模板

```bash
curl --location 'https://write.mcp.it.woa.com' \
--header 'Content-Type: application/json' \
--header "Authorization: Bearer $AI_WRITE_API_TOKEN" \
--data '{
    "version": 1,
    "eventId": "<uuid>",
    "componentName": "codebuddy",
    "timestamp": "<unix_timestamp>",
    "interface": {
        "interfaceName": "<METHOD>",
        "para": { <PARAMS> }
    }
}'
```

---

## 接口速查表

| 功能 | 接口名 | 核心参数 |
|------|--------|---------|
| 获取我创建的项目 | `write.solution.GetCreateSolutions.v1` | `page`, `PageSize` |
| 获取我加入的项目 | `write.solution.GetJoinSolutions.v1` | `page`, `PageSize` |
| 获取项目详情(含版本列表) | `write.solution.getSolutionInfo.v1` | `solutionId` |
| 获取文档目录树 | `write.node.GetTree.v1` | `solutionId`, `versionId`(可选) |
| 获取文档内容 | `write.article.detail.v1` | `nodeId`, `contentType:"markdown"` |
| 创建文档节点 | `write.node.AddNode.v1` | `solutionId`, `anchorType`, `nodeName`, `versionId`(可选) |
| 保存文档内容 | `write.article.save.v1` | `nodeId`, `content`, `contentType:"markdown"` |
| 锁定文档 | `write.article.lockArticle.v1` | `nodeId` |
| 解锁文档 | `write.article.unLockArticle.v1` | `nodeId` |
| 调整目录顺序 | `write.node.SortNode.v1` | `solutionId`, `tree` |
| 获取图片上传 URL | `write.inner.GetPresignedUploadUrl.v1` | `solutionId`, `suffix` |

> **重要**：读取和保存文档时，`contentType` 固定传 `"markdown"`，content 为标准 Markdown 文本。

---

## curl 示例

### 1. 获取项目列表

```bash
curl --location 'https://write.mcp.it.woa.com' \
--header 'Content-Type: application/json' \
--header "Authorization: Bearer $AI_WRITE_API_TOKEN" \
--data '{
    "version": 1,
    "eventId": "b1db18ba-1362-4be7-bd27-ea22a54f9aba",
    "componentName": "codebuddy",
    "timestamp": "1776168936",
    "interface": {
        "interfaceName": "write.solution.GetCreateSolutions.v1",
        "para": {
            "page": 1,
            "PageSize": 50
        }
    }
}'
```

### 1-b. 获取项目详情（含版本列表）

```bash
curl --location 'https://ai.write.woa.com' \
--header 'Content-Type: application/json' \
--header "Authorization: Bearer $AI_WRITE_API_TOKEN" \
--data '{
    "version": 1,
    "eventId": "solutioninfo-001",
    "componentName": "codebuddy",
    "timestamp": "1776168936",
    "interface": {
        "interfaceName": "write.solution.getSolutionInfo.v1",
        "para": {
            "solutionId": "78595548379652096"
        }
    }
}'
# 返回 data.versions[] 示例:
# [{"versionId":"2024001","versionName":"默认版本","lang":"zh","isDefault":1},
#  {"versionId":"2024002","versionName":"English","lang":"en","isDefault":0}]
```

### 2. 获取目录树

```bash
curl --location 'https://write.mcp.it.woa.com' \
--header 'Content-Type: application/json' \
--header "Authorization: Bearer $AI_WRITE_API_TOKEN" \
--data '{
    "version": 1,
    "eventId": "c2ec29cb-2473-5cf8-ce38-fb33b5g0abbc",
    "componentName": "codebuddy",
    "timestamp": "1776168936",
    "interface": {
        "interfaceName": "write.node.GetTree.v1",
        "para": {
            "solutionId": "78595548379652096"
        }
    }
}'
```

### 3. 获取文档内容（Markdown）

```bash
curl --location 'https://write.mcp.it.woa.com' \
--header 'Content-Type: application/json' \
--header "Authorization: Bearer $AI_WRITE_API_TOKEN" \
--data '{
    "version": 1,
    "eventId": "d3fd30dc-3584-6dg9-df49-gc44c6h1bced",
    "componentName": "codebuddy",
    "timestamp": "1776168936",
    "interface": {
        "interfaceName": "write.article.detail.v1",
        "para": {
            "nodeId": "78595548379652100",
            "contentType": "markdown"
        }
    }
}'
```

### 4. 创建文档节点

```bash
curl --location 'https://write.mcp.it.woa.com' \
--header 'Content-Type: application/json' \
--header "Authorization: Bearer $AI_WRITE_API_TOKEN" \
--data '{
    "version": 1,
    "eventId": "e4ge41ed-4695-7eh0-eg50-hd55d7i2cdfe",
    "componentName": "codebuddy",
    "timestamp": "1776168936",
    "interface": {
        "interfaceName": "write.node.AddNode.v1",
        "para": {
            "solutionId": "78595548379652096",
            "nodeName": "新文档标题",
            "anchorId": "78595548379652100",
            "anchorType": "down"
        }
    }
}'
```

### 5. 保存文档（完整流程：加锁→保存→解锁）

```bash
# Step 1: 加锁
curl --location 'https://write.mcp.it.woa.com' \
--header 'Content-Type: application/json' \
--header "Authorization: Bearer $AI_WRITE_API_TOKEN" \
--data '{
    "version": 1,
    "eventId": "lock-001",
    "componentName": "codebuddy",
    "timestamp": "1776168936",
    "interface": {
        "interfaceName": "write.article.lockArticle.v1",
        "para": {
            "nodeId": "78595548379652100"
        }
    }
}'

# Step 2: 保存 Markdown 内容
curl --location 'https://write.mcp.it.woa.com' \
--header 'Content-Type: application/json' \
--header "Authorization: Bearer $AI_WRITE_API_TOKEN" \
--data '{
    "version": 1,
    "eventId": "save-001",
    "componentName": "codebuddy",
    "timestamp": "1776168936",
    "interface": {
        "interfaceName": "write.article.save.v1",
        "para": {
            "nodeId": "78595548379652100",
            "contentType": "markdown",
            "content": "# 产品简介\n\n这是一篇新文档。\n\n## 功能特性\n\n- 功能一\n- 功能二"
        }
    }
}'

# Step 3: 解锁
curl --location 'https://write.mcp.it.woa.com' \
--header 'Content-Type: application/json' \
--header "Authorization: Bearer $AI_WRITE_API_TOKEN" \
--data '{
    "version": 1,
    "eventId": "unlock-001",
    "componentName": "codebuddy",
    "timestamp": "1776168936",
    "interface": {
        "interfaceName": "write.article.unLockArticle.v1",
        "para": {
            "nodeId": "78595548379652100"
        }
    }
}'
```

### 6. 调整目录顺序

```bash
curl --location 'https://write.mcp.it.woa.com' \
--header 'Content-Type: application/json' \
--header "Authorization: Bearer $AI_WRITE_API_TOKEN" \
--data '{
    "version": 1,
    "eventId": "sort-001",
    "componentName": "codebuddy",
    "timestamp": "1776168936",
    "interface": {
        "interfaceName": "write.node.SortNode.v1",
        "para": {
            "solutionId": "78595548379652096",
            "tree": [
                {
                    "nodeId": "123458",
                    "children": []
                },
                {
                    "nodeId": "123456",
                    "children": [
                        { "nodeId": "123457", "children": [] }
                    ]
                }
            ]
        }
    }
}'
```

### 7. 上传图片

```bash
# Step 1: 获取预签名上传 URL
curl -s --location 'https://write.mcp.it.woa.com' \
--header 'Content-Type: application/json' \
--header "Authorization: Bearer $AI_WRITE_API_TOKEN" \
--data '{
    "version": 1,
    "eventId": "upload-001",
    "componentName": "codebuddy",
    "timestamp": "1776168936",
    "interface": {
        "interfaceName": "write.inner.GetPresignedUploadUrl.v1",
        "para": {
            "solutionId": "112417317272764416",
            "suffix": ".png"
        }
    }
}'
# 返回: {"data": {"url": "https://xxx.cos.ap-guangzhou.myqcloud.com/...", "path": "123/uuid.png", "fileName": "uuid.png"}}

# Step 2: 用预签名 URL 上传图片
curl -X PUT "<返回的url>" --data-binary @image.png
```

---

## anchorType 参考

| 值 | 效果 | 是否需要 anchorId |
|----|------|:---:|
| `top` | 插入到目录顶部 | 否 |
| `up` | 在 anchorId 上方同级插入 | 是 |
| `down` | 在 anchorId 下方同级插入 | 是 |
| `sub_up` | 作为 anchorId 的第一个子节点 | 是 |
