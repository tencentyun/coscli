# SDK 文档写作模板

> 来源：SDK 文档写作模板（公有云官网平台需求）。
> 当前用户使用 SDK 文档的主要痛点为信息缺漏、步骤间跳跃大等问题，新手无法快速上手。本模板旨在解决该问题。

---

## SDK 文档结构

SDK 文档基本包含两大模块：
1. **初始化服务**：环境准备配置、安装 SDK、获取密钥、初始化配置
2. **SDK 实操**：按功能模块分篇，每篇包含功能描述、方法说明、完整示例、常见问题、API 操作链接

---

## 模板结构

### 第一部分：初始化服务

```
# [语言] SDK 快速入门

## 环境准备与配置

[介绍使用该 SDK 的环境要求、依赖项、环境的安装及配置。]

在开始执行操作前，您需要完成环境的安装及配置。

SDK 支持 [语言版本要求]，您可以通过命令 `[版本查看命令]` 查看版本。

[语言]安装方式请参见[安装文档链接]。

### 依赖项

```[语言]
[依赖配置示例]
```

## 安装 SDK

[介绍 SDK 安装及验证的操作步骤。]

1. 打开终端。
   - Windows：Win + R → 输入 cmd
   - Mac/Linux：打开 Terminal
   - IDE：在 VS Code/Goland 中按 Ctrl+` 调出终端
2. 进入项目目录。
   ```shell
   # 切换到项目根目录
   cd /path/to/your/project
   ```
3. 执行以下命令安装 SDK。
   ```[语言]
   [安装命令]
   ```
4. 验证安装。
   ```[语言]
   [验证命令]
   ```
   预期输出应包含以下，表示安装成功。
   ```
   [预期输出]
   ```

## 初始化服务

### 获取密钥

[介绍获取初始化 SDK 过程中使用的密钥。]

在开始执行操作前，您需要获取初始化 SDK 的密钥。推荐您使用临时密钥。

### 初始化配置

[介绍在使用服务相关请求前，需要实例化的对象。Step by Step 描述操作步骤，指导用户完成初始化。]

在执行任何和[产品名称]服务相关请求之前，都需要先实例化以下对象：

1. 执行以下命令，初始化服务配置。
   ```[语言]
   [初始化配置代码示例，带注释]
   ```

2. 提供访问凭证。
   SDK 中提供了多种方式：

   **方式1：持续更新的临时密钥（推荐）**
   ```[语言]
   [临时密钥代码示例]
   ```

   **方式2：不变的临时密钥**
   ```[语言]
   [临时密钥代码示例]
   ```

   **方式3：永久密钥**
   ```[语言]
   [永久密钥代码示例]
   ```

   > 注意：建议使用临时密钥，降低使用风险。

3. 创建服务实例。
   ```[语言]
   [创建服务实例代码示例]
   ```
```

### 第二部分：SDK 实操（每个功能一篇）

```
# [功能名称]

## 功能说明

[功能描述]

### 方法原型

```[语言]
[方法签名]
```

### 请求参数

| 参数名 | 类型 | 必填 | 描述 |
|-------|------|------|------|
| [参数名] | [类型] | [是/否] | [参数描述] |

### 返回结果

| 参数名 | 类型 | 描述 |
|-------|------|------|
| [参数名] | [类型] | [描述] |

### 完整示例

```[语言]
[完整的可运行代码示例，包含：
- 引入依赖
- 初始化
- 调用功能
- 处理结果
- 错误处理]
```

## 常见问题

### [问题一]？

[解决方案]

### [问题二]？

[解决方案]

## API 操作

[该功能通过 API 使用的相关文档链接。]

关于[操作名称]的 API 接口说明，请参见[API 文档链接]。
```

---

## 完整示例参考

### 初始化（COS Go SDK）

```go
package main

import (
    "context"
    "net/http"
    "net/url"

    "github.com/tencentyun/cos-go-sdk-v5"
)

func main() {
    // 替换为您的存储桶地域和名称
    u, _ := url.Parse("https://examplebucket-1250000000.cos.ap-guangzhou.myqcloud.com")
    b := &cos.BaseURL{BucketURL: u}
    client := cos.NewClient(b, &http.Client{
        Transport: &cos.AuthorizationTransport{
            SecretID:  "COS_SECRETID",  // 替换为您的 SecretId
            SecretKey: "COS_SECRETKEY", // 替换为您的 SecretKey
        },
    })

    // 使用 client 进行操作
    _ = client
}
```

### 功能实操（COS Python SDK 上传对象）

```python
# -*- coding=utf-8
from qcloud_cos import CosConfig
from qcloud_cos import CosS3Client
from qcloud_cos.cos_exception import CosClientError, CosServiceError
import sys
import os
import logging

# 正常情况日志级别使用 INFO，需要定位时可以修改为 DEBUG
logging.basicConfig(level=logging.INFO, stream=sys.stdout)

# 设置用户属性
secret_id = os.environ['COS_SECRET_ID']
secret_key = os.environ['COS_SECRET_KEY']
region = 'ap-beijing'
token = None
scheme = 'https'

config = CosConfig(Region=region, SecretId=secret_id,
                   SecretKey=secret_key, Token=token, Scheme=scheme)
client = CosS3Client(config)

# 使用高级接口上传
response = client.upload_file(
    Bucket='examplebucket-1250000000',
    Key='exampleobject',
    LocalFilePath='local.txt',
    EnableMD5=False,
    progress_callback=None
)

# 使用高级接口断点续传，失败重试时不会上传已成功的分块
for i in range(0, 10):
    try:
        response = client.upload_file(
            Bucket='examplebucket-1250000000',
            Key='exampleobject',
            LocalFilePath='local.txt')
        break
    except CosClientError or CosServiceError as e:
        print(e)
```

---

## 写作要点

1. **信息完整性**：每个步骤不能跳跃，新手能直接照着操作。
2. **代码可运行**：提供的代码示例必须可直接复制运行，敏感信息使用环境变量或占位符。
3. **多语言适配**：同一产品的 SDK 文档需覆盖多种语言（Python、Java、Go、Node.js、PHP、.NET 等），每种语言单独一篇。
4. **凭证安全**：推荐临时密钥，永久密钥方式需加安全提示。
5. **常见问题必选**：梳理用户在安装、配置环境等过程中可能遇到的典型问题并给出解决方案。
