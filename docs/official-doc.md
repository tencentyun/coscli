
# coscli 命令行工具

coscli 是腾讯云对象存储（Cloud Object Storage，COS）官方推出的命令行工具，用于便捷地管理存储桶（Bucket）和对象（Object）。coscli 基于 Go 语言开发，支持跨桶操作、分片上传/下载、断点续传、增量同步、软链接处理等能力，同时兼容标准 COS 桶和元数据加速桶（OFS）。

## 文档约定

- 命令示例以 `./coscli` 开头。Windows 环境下可替换为 `coscli.exe`。
- `cos://<bucket-alias>` 中的 `<bucket-alias>` 可以是配置文件中的桶别名（Alias），也可以是完整桶名（`<BucketName>-<AppID>` 形式）。
- `<appid>` 为腾讯云账号 APPID，是桶名的后缀部分（例如 `examplebucket-1250000000` 中的 `1250000000`）。
- OFS 桶：指开启了元数据加速能力的存储桶。部分命令在 OFS 桶上的行为与 COS 桶存在差异，已在文档中以「**OFS 差异**」单独说明。
- **`--include` / `--exclude` 过滤规则**：支持**标准正则表达式语法**（基于 Go [`regexp`](https://pkg.go.dev/regexp/syntax) 包，RE2 语法），**不支持 shell 通配符（glob）**。常见写法见下表：

| 需求 | 推荐写法（正则） | 常见错误写法（glob） |
|---|---|---|
| 匹配所有 `.log` 文件（后缀精确） | `'\.log$'` | ~~`'*.log'`~~ |
| 匹配 `.jpg` 或 `.png` 文件（多后缀） | `'\.(jpg`&#124;`png)$'` | ~~`'*.{jpg,png}'`~~ |
| 匹配 `logs/` 目录下所有对象（前缀锚定） | `'^logs/'` | ~~`'logs/*'`~~ |
| 匹配路径中包含 `/img/` 的对象（精确子路径） | `'/img/'` | ~~`'*/img/*'`~~ |
| 匹配以 `dirD` 开头的对象 | `'^dirD'` | — |

> **正则语法要点：**
>
> - `.` 匹配任意字符；要匹配字面量 `.`（如文件名中的 `.`），必须写成 `\.`。例如 `'.log$'` 会匹配 `xlog`（`.` 匹配 `x`），而 `'\.log$'` 只匹配真正以 `.log` 结尾的文件。
> - `*` 表示"前一个字符/分组重复 0 次或多次"，**`*` 前必须有字符**；`*.log`、`*abc`、`+foo` 等"以量词开头"的模式都是**非法正则**。
> - `^` 锚定字符串开头，`$` 锚定字符串结尾；两者都省略时采用**部分匹配**（子串包含即命中），例如 `'log'` 会同时匹配 `mylog.txt` 和 `log.go`、`catalog` 等。
> - 多选用 `|`（竖线），两种等价写法：
>   - `'\.jpg$|\.png$'`：分支并列，每个分支独立带锚点。
>   - `'\.(jpg|png)$'`：**分组并列（更简洁，推荐）**。
> - 支持简写 `[!abc]`，工具会自动预处理为 `[^abc]`（匹配非 `a`/`b`/`c` 的字符）。

> **⚠️ 过滤匹配的"对象"是什么？（容易踩坑）**
>
> 过滤规则作用于字符串，但**不同命令传入的字符串不同**，这是最容易被忽略的差异：
>
> | 场景 | 匹配对象 | 示例输入 |
> |---|---|---|
> | **上传**（`cp` / `sync` 本地→COS） | **本地相对路径** | Linux/macOS：`logs/app.log`；Windows：`logs\app.log` |
> | **下载**（`cp` / `sync` COS→本地） | **COS 对象 Key**（完整 Key） | `logs/2024/app.log` |
> | **拷贝**（`cp` / `sync` COS→COS） | **源桶的对象 Key** | `src/path/a.log` |
> | **列举/删除/恢复**（`ls` / `rm` / `restore` / `du` / `lsdu` / `abort`） | **对象 Key** 或 **CommonPrefix**（目录前缀） | `dir/sub/a.log`，`dir/sub/` |
>
> **关键结论：**
>
> 1. 对 COS 侧命令，匹配的是 **完整 Key**（不含桶名，但含所有目录前缀），不是"当前路径下的相对名"。例如在 `cos://bkt/data/` 下执行 `ls --include '\.log$'`，实际匹配的是 `data/file.log` 这种完整 Key。
> 2. 对上传场景，**Windows 下路径分隔符是 `\`**，正则中写 `'^logs/'` 在 Windows 上不生效，需要写成 `'^logs[/\\]'` 以兼容两种平台。
> 3. 无锚点的简写易误命中：`'img'` 会同时匹配 `data/img/a.png` 和 `data/images/b.jpg`，建议用 `/img/` 或 `^img/` 精确表达意图。

> **⚠️ 非法正则的静默行为（容易踩坑）：**
>
> 当传入非法正则（如 `'*.log'`）时，工具**不会报错也不会给出提示**，但在 `include` 和 `exclude` 两种模式下行为**不对称**：
>
> | 参数 | 正则合法时 | **正则非法时（静默）** |
> |---|---|---|
> | `--include '*.log'` | 仅保留匹配对象 | **过滤结果为空**，所有对象都被过滤掉 |
> | `--exclude '*.log'` | 排除匹配对象 | **过滤完全失效**，所有对象都通过 |
>
> 如果执行结果与预期不符（例如"上传了 0 个文件"或"exclude 没生效"），请**先检查正则是否合法**。可通过 Go Playground 或本地脚本 `regexp.MatchString(pattern, "test-key")` 预先验证。

> **Shell 转义建议：**
>
> - **强烈建议使用单引号包裹正则**（`'...'`），shell 会原样保留引号内的 `\`、`$`、`|` 等字符。
> - 双引号下 bash 对 `\` 的处理：`"\."` 原样保留为 `\.`（因为 `.` 不是 bash 特殊字符），`"\\."` 会被 bash 转为 `\.`，两种写法传给 Go 的都是 `\.`，**功能等效但单引号可读性最好**。
> - Windows `cmd` 下若无法使用单引号，可改用双引号并将反斜杠写成 `\\`，例如 `--include ".*\\.log$"`。

## 前提条件

- 已在腾讯云访问管理控制台获取到用户的 SecretId 和 SecretKey。
- 已在 COS 控制台创建存储桶。
- 本地环境为 Linux、macOS 或 Windows。

## 目录

- [一、全局参数](#一全局参数)
- [二、配置管理](#二配置管理)
  - [2.1 config init（交互式初始化）](#21-config-init)
  - [2.2 config add（添加桶）](#22-config-add)
  - [2.3 config set（修改 base 配置）](#23-config-set)
  - [2.4 config delete（删除桶）](#24-config-delete)
  - [2.5 config show（查看配置）](#25-config-show)
- [三、桶操作](#三桶操作)
  - [3.1 mb（创建桶）](#31-mb创建桶)
  - [3.2 rb（删除桶）](#32-rb删除桶)
  - [3.3 ls（列出桶或对象）](#33-ls列出桶或对象)
  - [3.4 du（按存储类型统计）](#34-du按存储类型统计)
  - [3.5 lsdu（按子目录统计）](#35-lsdu按子目录统计)
  - [3.6 stat（查询对象元信息）](#36-stat查询对象元信息)
- [四、对象上传下载与复制](#四对象上传下载与复制)
  - [4.1 cp（上传/下载/复制）](#41-cp上传下载复制)
  - [4.2 sync（增量同步）](#42-sync增量同步)
  - [4.3 rm（删除对象）](#43-rm删除对象)
  - [4.4 cat（查看对象内容）](#44-cat查看对象内容)
- [五、对象高级操作](#五对象高级操作)
  - [5.1 hash（计算哈希值）](#51-hash计算哈希值)
  - [5.2 signurl（生成预签名 URL）](#52-signurl生成预签名-url)
  - [5.3 restore（恢复归档对象）](#53-restore恢复归档对象)
  - [5.4 symlink（软链接管理）](#54-symlink软链接管理)
  - [5.5 abort（终止分片上传）](#55-abort终止分片上传)
  - [5.6 lsparts（列出分片上传任务）](#56-lsparts列出分片上传任务)
- [六、桶配置管理](#六桶配置管理)
  - [6.1 bucket-acl（桶 ACL）](#61-bucket-acl桶-acl)
  - [6.2 bucket-encryption（桶加密）](#62-bucket-encryption桶加密)
  - [6.3 inventory（清单任务）](#63-inventory清单任务)
  - [6.4 bucket-policy（桶策略）](#64-bucket-policy桶策略)
  - [6.5 bucket-tagging（桶标签）](#65-bucket-tagging桶标签)
  - [6.6 bucket-versioning（桶版本控制）](#66-bucket-versioning桶版本控制)
- [七、对象属性管理](#七对象属性管理)
  - [7.1 object-acl（对象 ACL）](#71-object-acl对象-acl)
  - [7.2 object-tagging（对象标签）](#72-object-tagging对象标签)
- [八、常见场景示例](#八常见场景示例)
- [九、COS 与 OFS 差异汇总](#九cos-与-ofs-差异汇总)
- [十、配置文件参考](#十配置文件参考)
- [十一、退出码](#十一退出码)
- [十二、参考资料](#十二参考资料)

## 一、全局参数

以下参数对所有命令生效（通过 `rootCmd.PersistentFlags()` 注册），可在任何子命令后追加。

| 参数 | 短参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|---|
| `--config-path` | `-c` | String | `$HOME/.cos.yaml` | 指定配置文件路径。 |
| `--secret-id` | `-i` | String | 空 | 临时指定 SecretId，覆盖配置文件中的 `cos.base.secretid`。 |
| `--secret-key` | `-k` | String | 空 | 临时指定 SecretKey，覆盖配置文件中的 `cos.base.secretkey`。 |
| `--token` | 无 | String | 空 | 临时指定 SessionToken（使用临时密钥时必填）。 |
| `--endpoint` | `-e` | String | 空 | 临时指定请求域名，覆盖配置文件中的桶 Endpoint。 |
| `--customized` | 无 | Bool | `false` | 开启自定义域名模式。开启后，`--endpoint` 的值将作为完整访问域名，而不会拼接桶名。 |
| `--protocol` | `-p` | String | `https` | 请求协议，可选 `https` 或 `http`。 |
| `--init-skip` | 无 | Bool | `false` | 配置文件不存在时跳过交互式初始化流程。 |
| `--log-path` | 无 | String | 可执行文件同目录 | 指定日志目录，日志文件名为 `coscli.log`。 |
| `--disable-log` | 无 | Bool | `false` | 关闭日志输出。 |
| `--close_auto_switch_host` | 无 | String | 空 | 关闭自动切换备用域名。设置为 `"true"` 表示关闭。 |
| `--bucket-type` | 无 | String | 空 | 显式指定桶类型，可选 `COS` 或 `OFS`，避免每次请求自动探测。 |
| `--proxy` | 无 | String | 空 | 指定代理地址，例如 `http://<username>:<password>@host:port` 或 `socks5://host:port`。 |
| `--version` | `-v` | — | — | 输出 coscli 版本号后退出。 |
| `--help` | `-h` | — | — | 输出帮助信息后退出。可在任意子命令后使用（如 `./coscli cp -h`）查看该子命令的所有参数。 |

**示例：**

```shell
# 使用指定配置文件
./coscli ls cos://examplebucket -c /data/my.cos.yaml

# 临时指定密钥与 Endpoint
./coscli ls cos://examplebucket-1250000000 \
    -i AKIDxxxxxxxx -k xxxxxxxx \
    -e cos.ap-guangzhou.myqcloud.com

# 使用代理访问 COS
./coscli cp ./local.txt cos://examplebucket/ --proxy http://127.0.0.1:8080

# 查看版本
./coscli -v

# 查看子命令帮助
./coscli cp -h
```

**输出示例（`-v`）：**

```text
coscli version v1.0.8
```

## 二、配置管理

### 2.1 config init

用于交互式生成配置文件。首次使用 coscli 时会自动进入该流程。

**语法格式：**

```shell
./coscli config init [-c <config-file-path>]
```

**参数说明：**

该命令无专属参数，仅支持全局参数 `-c`。交互过程中会依次提示以下输入项：

| 输入项 | 说明 |
|---|---|
| 配置文件路径 | 默认为 `$HOME/.cos.yaml`，可自定义路径。 |
| Mode | 鉴权模式，填 `SecretKey` 或 `CvmRole`。 |
| Cvm Role Name | 当 Mode 为 `CvmRole` 时填写，使用 CVM 实例角色鉴权。 |
| Secret ID / Secret Key / Session Token | 当 Mode 为 `SecretKey` 时填写。 |
| DisableEncryption | 是否禁用密钥加密存储。填 `true`/`false`，默认 `false`。 |
| DisableAutoFetchBucketType | 是否禁用自动获取桶类型。填 `true`/`false`，默认 `false`。 |
| CloseAutoSwitchHost | 是否关闭自动切换备用域名。填 `true`/`false`，默认 `false`。 |
| Bucket Name | 桶名，格式 `<bucketname>-<appid>`。 |
| Bucket Endpoint | 桶的访问域名，例如 `cos.ap-guangzhou.myqcloud.com`。 |
| Bucket Alias | 桶别名，留空则使用桶名。 |
| Customized | 是否使用自定义域名，填 `true`/`false`，默认 `false`。 |

**示例：**

```shell
./coscli config init
```

**交互输出示例：**

```text
Specify the path of the configuration file: (default:/root/.cos.yaml)

The path of the configuration file: /root/.cos.yaml
Input Your Mode:
SecretKey
Input Your Secret ID:
AKIDxxxxxxxxxxxxxxxxxx
Input Your Secret Key:
********
Input Your Session Token:

Disable encryption (DisableEncryption)? (true/false, default: false):

Disable automatic bucket type fetching (DisableAutoFetchBucketType)? (true/false, default: false):

Disable automatic backup domain switching (CloseAutoSwitchHost)? (true/false, default: false):

Input Your Bucket's Name:
Format: <bucketname>-<appid>，Example: example-1234567890
examplebucket-1250000000
Input Bucket's Endpoint:
Format: cos.<region>.myqcloud.com，Example: cos.ap-beijing.myqcloud.com
cos.ap-guangzhou.myqcloud.com
Input Bucket's Alias: (Input nothing will use the original name)
example
Use customized endpoint for this bucket? (true/false, default: false):

You have configured the bucket:
- Name: examplebucket-1250000000    Endpoint: cos.ap-guangzhou.myqcloud.com    Alias: example

If you want to configure more buckets, you can use the "config add" command later.

The configuration file is initialized successfully!
You can use "./coscli config show [-c <Config File Path>]" show the contents of the specified configuration file
```

### 2.2 config add

用于在现有配置文件中新增一个存储桶配置。

**语法格式：**

```shell
./coscli config add -b <bucket-name> [-e <endpoint>] [-r <region>] [-a <alias>] [-o] [--customized]
```

**参数说明：**

| 参数 | 短参数 | 类型 | 默认值 | 必填 | 说明 |
|---|---|---|---|---|---|
| `--bucket` | `-b` | String | 空 | 是 | 桶的完整名称，格式为 `<BucketName>-<AppID>`。 |
| `--endpoint` | `-e` | String | 空 | 否 | 桶的访问域名。未填写时会根据 `--region` 自动推导为 `cos.<region>.myqcloud.com`。 |
| `--region` | `-r` | String | 空 | 否 | 桶所属地域，例如 `ap-guangzhou`。 |
| `--alias` | `-a` | String | 与 `--bucket` 一致 | 否 | 桶别名。别名不可与其他桶的 Name 或 Alias 重复。 |
| `--ofs` | `-o` | Bool | `false` | 否 | 标记为 OFS（元数据加速）桶。标准 COS 桶保持 `false`。 |
| `--customized` | 无 | Bool | `false` | 否 | 启用该桶的自定义域名模式。开启后 `--endpoint` 将作为完整访问域名。 |

**示例：**

```shell
# 添加标准 COS 桶
./coscli config add -b examplebucket-1250000000 -r ap-guangzhou -a example

# 添加 OFS 桶
./coscli config add -b ofsbucket-1250000000 -r ap-guangzhou -a ofs-example -o

# 使用自定义域名
./coscli config add -b examplebucket-1250000000 \
    -e my-cdn-domain.example.com \
    -a example --customized
```

**输出示例：**

```text
INFO[2026-04-22 15:20:00] Add successfully! name: examplebucket-1250000000, endpoint: cos.ap-guangzhou.myqcloud.com, alias: example, ofs: false, customized: false 
```

### 2.3 config set

用于修改配置文件 `base` 段的全局配置项。

**语法格式：**

```shell
./coscli config set [flags]
```

**参数说明：**

| 参数 | 短参数 | 类型 | 说明 |
|---|---|---|---|
| `--secret_id` | 无 | String | 修改 SecretId。传 `@` 表示清空该项。 |
| `--secret_key` | 无 | String | 修改 SecretKey。传 `@` 表示清空该项。 |
| `--session_token` | `-t` | String | 修改 SessionToken。传 `@` 表示清空该项。 |
| `--mode` | 无 | String | 鉴权模式，仅允许 `SecretKey` 或 `CvmRole`。 |
| `--cvm_role_name` | 无 | String | 指定 CVM 实例角色名称。传 `@` 表示清空该项。 |
| `--close_auto_switch_host` | 无 | String | 关闭自动切换备用域名。传 `"true"` 或 `"false"`，传 `@` 清空。 |
| `--disable_encryption` | 无 | String | 是否禁用密钥加密存储。传 `"true"` 或 `"false"`，传 `@` 清空。 |
| `--disable_auto_fetch_bucket_type` | 无 | String | 是否禁用自动获取桶类型。传 `"true"` 或 `"false"`，传 `@` 清空。 |
| `--proxy` | 无 | String | 设置代理地址。传 `@` 表示清空该项。 |

> **说明：** 至少需要指定一个参数，否则命令返回错误。

**示例：**

```shell
# 更新临时会话令牌
./coscli config set -t examplesessiontoken

# 关闭密钥加密存储
./coscli config set --disable_encryption true

# 清空代理设置
./coscli config set --proxy @
```

**输出示例：**

```text
INFO[2026-04-22 15:21:00] Modify successfully!                         
```

### 2.4 config delete

用于从配置文件中删除一个存储桶配置。

**语法格式：**

```shell
./coscli config delete -a <alias>
```

**参数说明：**

| 参数 | 短参数 | 类型 | 必填 | 说明 |
|---|---|---|---|---|
| `--alias` | `-a` | String | 是 | 要删除的桶别名。 |

**示例：**

```shell
./coscli config delete -a example
```

**输出示例：**

```text
INFO[2026-04-22 15:22:00] Delete successfully! name: examplebucket-1250000000, endpoint: cos.ap-guangzhou.myqcloud.com, alias: example 
```

### 2.5 config show

用于查看当前配置文件的内容。

**语法格式：**

```shell
./coscli config show [-c <config-file-path>]
```

该命令无专属参数，仅支持全局参数 `-c`。输出包含基础配置和所有已配置桶的详细信息。

**示例：**

```shell
./coscli config show
```

**输出示例：**

> **说明：** 桶信息中 `Name`、`Endpoint`、`Alias`、`Ofs`、`Customized` 等字段后使用制表符（`\t`）对齐，不是等宽空格。

```text
Configuration file path:
  /root/.cos.yaml
====================
Basic Configuration Information:
  Secret ID:     AKIDxxxxxxxxxxxxxxxxxx
  Secret Key:    **加密存储，不显示明文**
  Session Token: 
  Mode: SecretKey
  CvmRoleName: 
  CloseAutoSwitchHost: 
  DisableEncryption: 
  DisableAutoFetchBucketType: 
  Proxy: 
====================
Bucket Configuration Information:
- Bucket 1 :
  Name:  	examplebucket-1250000000
  Endpoint:	cos.ap-guangzhou.myqcloud.com
  Alias: 	example
  Ofs: 	false
  Customized:	false
```

## 三、桶操作

### 3.1 mb（创建桶）

用于创建存储桶。

**语法格式：**

```shell
./coscli mb cos://<bucket-name>-<appid> -e <endpoint> [flags]
```

**参数说明：**

| 参数 | 短参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|---|
| `--region` | `-r` | String | 空 | 桶所属地域。当未指定 `--endpoint` 时，系统会将其自动拼接为 `cos.<region>.myqcloud.com`。 |
| `--ofs` | `-o` | Bool | `false` | 创建 OFS（元数据加速）桶。 |
| `--maz` | `-m` | Bool | `false` | 创建多 AZ 桶。 |
| `--acl` | 无 | String | 空 | 桶访问权限。可选 `private`、`public-read`、`public-read-write`、`authenticated-read`。 |
| `--grant-read` | 无 | String | 空 | 授予指定账号读权限，格式 `id="100000000001",id="100000000002"`。 |
| `--grant-write` | 无 | String | 空 | 授予指定账号写权限。 |
| `--grant-read-acp` | 无 | String | 空 | 授予指定账号读 ACL 权限。 |
| `--grant-write-acp` | 无 | String | 空 | 授予指定账号写 ACL 权限。 |
| `--grant-full-control` | 无 | String | 空 | 授予指定账号完全控制权限。 |
| `--tags` | 无 | String | 空 | 桶标签，最多 10 个，格式 `Key1=Value1&Key2=Value2`，Key 和 Value 需先进行 URL 编码。 |

**所需权限：**

| 场景 | 权限 Action |
|---|---|
| 创建桶（含 ACL、标签、地域、桶类型等所有参数） | `cos:PutBucket` |

> **说明：** ACL 与标签通过创建桶请求的 HTTP 头部一次下发，仅需 `cos:PutBucket` 权限即可，无需单独授予 `cos:PutBucketACL` 或 `cos:PutBucketTagging`。

**示例：**

```shell
# 创建标准 COS 桶
./coscli mb cos://examplebucket-1250000000 -e cos.ap-guangzhou.myqcloud.com

# 创建 OFS 桶
./coscli mb cos://ofsbucket-1250000000 -r ap-guangzhou -o

# 创建多 AZ 桶，带标签
./coscli mb cos://examplebucket-1250000000 -r ap-guangzhou -m \
    --tags "env=prod&team=cos"
```

**输出示例：**

```text
INFO[2026-04-22 15:23:00] Create a new bucket! name: examplebucket-1250000000 
```

### 3.2 rb（删除桶）

用于删除存储桶。

**语法格式：**

```shell
./coscli rb cos://<bucket-name>-<appid> [flags]
```

**参数说明：**

| 参数 | 短参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|---|
| `--force` | `-f` | Bool | `false` | 强制删除。开启后会先清空桶内所有对象（含历史版本、未完成分片），再删除桶。 |
| `--region` | `-r` | String | 空 | 桶所属地域，用于推导 Endpoint。 |
| `--fail-output` | 无 | Bool | `true` | 是否将删除失败的对象信息输出到文件。 |
| `--fail-output-path` | 无 | String | `coscli_output` | 删除失败信息的输出目录。 |

> **说明：** 无论是否使用 `--force`，都会交互式提示确认。

**OFS 差异：**

- 强制删除 OFS 桶时，不会清理历史版本（OFS 桶不支持多版本）。

**所需权限：**

| 场景 | 权限 Action |
|---|---|
| 删除空桶 | `cos:DeleteBucket` |
| 强制删除（`-f`）清理桶内对象 | `cos:HeadBucket`、`cos:GetBucket`、`cos:DeleteObject`、`cos:DeleteMultipleObjects` |
| 强制删除时清理未完成分片 | `cos:ListMultipartUploads`、`cos:AbortMultipartUpload` |
| 强制删除版本控制桶的历史版本 | `cos:GetBucketVersioning`、`cos:GetBucketObjectVersions`、`cos:DeleteObjectVersion` |

**示例：**

```shell
# 仅删除空桶
./coscli rb cos://examplebucket-1250000000 -e cos.ap-guangzhou.myqcloud.com

# 强制清空后删除
./coscli rb cos://examplebucket-1250000000 -r ap-guangzhou -f
```

**输出示例（删除空桶）：**

```text
INFO[2026-04-22 15:24:00] Do you want to delete examplebucket-1250000000? (y/n) 
y
INFO[2026-04-22 15:24:02] Delete a empty bucket! name: examplebucket-1250000000 
```

**输出示例（`-f` 强制清桶）：**

```text
INFO[2026-04-22 15:24:00] Do you want to clear all inside the bucket and delete bucket examplebucket-1250000000 ? (y/n) 
y
INFO[2026-04-22 15:24:03] Start remove prefix examplebucket-1250000000 
delete object count:12
INFO[2026-04-22 15:24:08] Remove prefix examplebucket-1250000000 completed 
INFO[2026-04-22 15:24:10] Delete a empty bucket! name: examplebucket-1250000000 
```

### 3.3 ls（列出桶或对象）

用于列出所有桶，或列出指定桶/前缀下的对象。

**语法格式：**

```shell
# 列出所有桶
./coscli ls

# 列出桶内对象
./coscli ls cos://<bucket-alias>[/prefix/] [flags]
```

**参数说明：**

| 参数 | 短参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|---|
| `--limit` | 无 | Int | `10000` | 限制返回条数，取值 `-1`（不限制）或大于 0 的整数。 |
| `--recursive` | `-r` | Bool | `false` | 递归列出所有子目录对象。 |
| `--include` | 无 | String | 空 | 包含过滤规则，仅展示匹配的对象。**支持标准正则表达式**，详见《文档约定》。 |
| `--exclude` | 无 | String | 空 | 排除过滤规则，不展示匹配的对象。**支持标准正则表达式**。 |
| `--all-versions` | 无 | Bool | `false` | 列出对象的所有历史版本（需桶已启用版本控制）。 |

**OFS 差异：**

- OFS 桶**不支持** `--all-versions`，指定时会返回错误。

**所需权限：**

| 场景 | 权限 Action |
|---|---|
| 列出账号下所有桶（无参执行） | `cos:GetService` |
| 列出桶内对象 | `cos:HeadBucket`、`cos:GetBucket` |
| 列出对象历史版本（`--all-versions`） | `cos:HeadBucket`、`cos:GetBucketVersioning`、`cos:GetBucketObjectVersions` |

> **说明：** 指定桶操作时，工具会优先调用 `HEAD Bucket` 探测桶类型（COS/OFS），故所有指定桶的 `ls` 操作均需 `cos:HeadBucket`。

**示例：**

```shell
# 列出账号下所有桶
./coscli ls

# 列出桶根目录（最多 10000 条）
./coscli ls cos://examplebucket

# 递归列出前缀 test/ 下全部对象
./coscli ls cos://examplebucket/test/ -r

# 仅列出 .jpg 文件（正则写法，单引号避免 shell 转义）
./coscli ls cos://examplebucket -r --include '\.jpg$'

# 列出对象的所有版本
./coscli ls cos://examplebucket -r --all-versions
```

**输出示例（列出所有桶）：**

> **说明：** 表格由 [tablewriter](https://github.com/olekukonko/tablewriter) 渲染，列宽自适应内容，使用 `---+---` 风格的分隔符。

```text
                   BUCKET NAME          |    REGION    |     CREATE DATE       
----------------------------------------+--------------+-----------------------
  examplebucket-1250000000              | ap-guangzhou | 2026-01-10T08:32:00Z  
  logbucket-1250000000                  | ap-shanghai  | 2026-02-15T11:05:22Z  
```

**输出示例（列出对象）：**

```text
              KEY              |   TYPE   |       LAST MODIFIED       |                ETAG                |      SIZE       | RESTORESTATUS  
-------------------------------+----------+---------------------------+------------------------------------+-----------------+----------------
  test/                        | DIR      |                           |                                    |                 |                
  test/a.txt                   | STANDARD | 2026-04-22T10:30:00+08:00 | "e0323a9039add2978bf5b49550572c7c" | 12.00 B         |                
  test/b.log                   | STANDARD | 2026-04-22T10:31:00+08:00 | "5d41402abc4b2a76b9719d911017c592" | 5.25 KB         |                
-------------------------------+----------+---------------------------+------------------------------------+-----------------+----------------
                                                                                                           TOTAL OBJECTS:  |       2        
                                                                                                         ------------------+----------------
```

### 3.4 du（统计存储大小、按存储类型分类）

按存储类型分类统计前缀下对象的数量和大小，结果输出包含每个存储类型的对象数和总大小，以及所有类型汇总值。

**语法格式：**

```shell
./coscli du cos://<bucket-alias>[/prefix/] [flags]
```

**参数说明：**

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| `--include` | String | 空 | 包含过滤规则。**支持标准正则表达式**，详见《文档约定》。 |
| `--exclude` | String | 空 | 排除过滤规则。**支持标准正则表达式**。 |
| `--all-versions` | Bool | `false` | 统计所有历史版本（需桶已启用版本控制）。 |

**OFS 差异：**

- OFS 桶不支持 `--all-versions`。

**所需权限：**

| 场景 | 权限 Action |
|---|---|
| 统计桶/前缀容量 | `cos:HeadBucket`、`cos:GetBucket` |
| 统计所有历史版本（`--all-versions`） | `cos:HeadBucket`、`cos:GetBucketVersioning`、`cos:GetBucketObjectVersions` |

**示例：**

```shell
# 按存储类型统计桶容量
./coscli du cos://examplebucket

# 统计指定前缀容量（含所有历史版本）
./coscli du cos://examplebucket/logs/ --all-versions

# 仅统计 .log 文件（正则写法，单引号避免 shell 转义）
# 注意：匹配对象是完整 Key（如 logs/app.log）
./coscli du cos://examplebucket/logs/ --include '\.log$'
```

**输出示例：**

> **说明：** 即使某些存储类型的对象数为 0，表格仍会完整列出 COS 支持的所有存储类型。

```text
       STORAGE CLASS      | OBJECTS COUNT | TOTAL SIZE  
--------------------------+---------------+-------------
                 STANDARD |           128 | 256.30 MB   
              STANDARD_IA |            42 | 1.02 GB     
      INTELLIGENT_TIERING |             0 | 0  B        
                  ARCHIVE |            15 | 512.00 MB   
             DEEP_ARCHIVE |             0 | 0  B        
             MAZ_STANDARD |             0 | 0  B        
          MAZ_STANDARD_IA |             0 | 0  B        
  MAZ_INTELLIGENT_TIERING |             0 | 0  B        
              MAZ_ARCHIVE |             0 | 0  B        
--------------------------+---------------+-------------
INFO[2026-04-22 15:25:00] Total Objects Count: 185                     
INFO[2026-04-22 15:25:00] Total Objects Size:  1.79 GB                 
```

### 3.5 lsdu（统计存储大小、按子目录分类）

按一级子目录分类统计前缀下对象的数量和大小，适用于快速定位占据容量最大的子目录。与 `du` 的区别在于：`du` 按存储类型聚合，`lsdu` 按一级子目录聚合。

**语法格式：**

```shell
./coscli lsdu cos://<bucket-alias>[/prefix/] [flags]
```

**参数说明：**

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| `--include` | String | 空 | 包含过滤规则。**支持标准正则表达式**，详见《文档约定》。 |
| `--exclude` | String | 空 | 排除过滤规则。**支持标准正则表达式**。 |

> **说明：** `lsdu` 不支持 `--all-versions`。

**所需权限：**

| 场景 | 权限 Action |
|---|---|
| 统计指定前缀下子目录的容量和对象数 | `cos:HeadBucket`、`cos:GetBucket` |

**示例：**

```shell
# 列出各子目录的容量占比
./coscli lsdu cos://examplebucket/data/

# 仅统计含 /img/ 子路径的对象（用精确子路径而非子串，避免误命中 /imgs/）
./coscli lsdu cos://examplebucket/data/ --include '/img/'
```

**输出示例：**

```text
      NAME      | OBJECTS COUNT | TOTAL SIZE  
----------------+---------------+-------------
  data/img/     |            58 | 150.20 MB   
  data/log/     |            23 | 45.10 MB    
  data/csv/     |            12 | 8.55 MB     
----------------+---------------+-------------
INFO[2026-04-22 15:26:00] Total Objects Count: 93                      
INFO[2026-04-22 15:26:00] Total Objects Size:  203.85 MB               
```

### 3.6 stat（查询对象元信息）

查询单个对象的元数据（包含 ETag、Content-Type、大小、最后修改时间、存储类型、版本 ID、自定义元信息等）。

**语法格式：**

```shell
./coscli stat cos://<bucket-alias>/<key> [--version-id <id>]
```

**参数说明：**

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| `--version-id` | String | 空 | 指定对象的版本 ID（桶已启用版本控制时可用）。 |

输出字段包括：`ETag`、`Content-Type`、`Content-Length`、`Last-Modified`、`Cache-Control`、`Content-Disposition`、`Content-Encoding`、`Content-Language`、`Expires`、`x-cos-storage-class`、`x-cos-version-id`、`x-cos-object-type`、`x-cos-hash-crc64ecma`、自定义元信息（`x-cos-meta-*`）。

**所需权限：**

| 场景 | 权限 Action |
|---|---|
| 查询对象元信息 | `cos:HeadObject` |
| 查询指定版本的元信息（`--version-id`） | `cos:HeadObject`（需桶已启用版本控制） |

**示例：**

```shell
./coscli stat cos://examplebucket-1250000000/test.txt
./coscli stat cos://examplebucket-1250000000/test.txt --version-id MTg0NDY3NDI1NTk5MjQ4OTA2NA
```

**输出示例：**

```text
INFO[2026-04-22 15:27:00] Object: cos://examplebucket-1250000000/test.txt 
INFO[2026-04-22 15:27:00]   ETag:                 "e0323a9039add2978bf5b49550572c7c" 
INFO[2026-04-22 15:27:00]   Content-Type:         text/plain           
INFO[2026-04-22 15:27:00]   Content-Length:       12                   
INFO[2026-04-22 15:27:00]   Last-Modified:        Wed, 22 Apr 2026 07:27:00 GMT 
INFO[2026-04-22 15:27:00]   Cache-Control:        max-age=3600         
INFO[2026-04-22 15:27:00]   Content-Disposition:                       
INFO[2026-04-22 15:27:00]   Content-Encoding:                          
INFO[2026-04-22 15:27:00]   Content-Language:                          
INFO[2026-04-22 15:27:00]   Expires:                                   
INFO[2026-04-22 15:27:00]   x-cos-storage-class:  STANDARD             
INFO[2026-04-22 15:27:00]   x-cos-version-id:     MTg0NDY3NDI1NTk5MjQ4OTA2NA 
INFO[2026-04-22 15:27:00]   x-cos-object-type:    normal               
INFO[2026-04-22 15:27:00]   x-cos-hash-crc64ecma: 10962195253874765732 
```

## 四、对象上传、下载与复制

### 4.1 cp（上传/下载/复制）

`cp` 命令根据源路径与目的路径的前缀自动识别操作类型：

| 源路径 | 目的路径 | 操作 |
|---|---|---|
| 本地路径 | `cos://...` | 上传 |
| `cos://...` | 本地路径 | 下载 |
| `cos://...` | `cos://...` | 拷贝 / 迁移（`--move`） |

**语法格式：**

```shell
./coscli cp <source_path> <destination_path> [flags]
```

**通用参数：**

| 参数 | 短参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|---|
| `--recursive` | `-r` | Bool | `false` | 递归处理目录。 |
| `--include` | 无 | String | 空 | 包含规则（需与 `-r` 配合）。**支持标准正则表达式**，详见《文档约定》。 |
| `--exclude` | 无 | String | 空 | 排除规则（需与 `-r` 配合）。**支持标准正则表达式**。 |
| `--storage-class` | 无 | String | 空 | 存储类型，可选 `STANDARD`、`STANDARD_IA`、`INTELLIGENT_TIERING`、`MAZ_STANDARD`、`MAZ_STANDARD_IA`、`ARCHIVE`、`DEEP_ARCHIVE`。下载时设置会返回错误。 |
| `--rate-limiting` | 无 | Float32 | `0` | 单链接限速，单位 MB/s。推荐取值区间 `0.1-100`；`0` 表示不限速。与 `--thread-num` 叠加后的总速度 = `--thread-num` × `--rate-limiting`。 |
| `--part-size` | 无 | Int64 | `32` | 分片大小（MB），最大支持 `5120`。设为 `0` 时根据文件大小自适应分块。 |
| `--thread-num` | 无 | Int | `0` | 分片并发线程数。`0` 表示按文件大小自动推算；`>0` 时覆盖自动推算且忽略 `--max-thread-num`。 |
| `--max-thread-num` | 无 | Int | `32` | `--thread-num` 为 `0` 时的自动推算上限。`--thread-num` 已显式指定时该参数无效。 |
| `--routines` | 无 | Int | `3` | 文件并发数（同时处理几个文件）。 |
| `--fail-output` | 无 | Bool | `true` | 是否将失败信息输出到文件。 |
| `--fail-output-path` | 无 | String | `coscli_output` | 失败输出目录。 |
| `--process-log` | 无 | Bool | `true` | 是否开启进程日志。 |
| `--process-log-path` | 无 | String | `coscli_output` | 进程日志输出目录。 |
| `--meta` | 无 | String | 空 | 设置对象元信息，格式 `header:value#header:value`，示例 `Cache-Control:no-cache#Content-Encoding:gzip`。 |
| `--retry-num` | 无 | Int | `0` | 限流重试次数，取值 `0-100`（**注意：官网旧文档描述为 `1-10` 是错误的**）。`0` 表示不重试。 |
| `--err-retry-num` | 无 | Int | `5` | 错误重试次数，取值 `0-100`。`0` 表示不重试。 |
| `--err-retry-interval` | 无 | Int | `0` | 错误重试间隔（秒），取值 `0-10`。`0` 表示每次随机 1-10 秒。 |
| `--disable-crc64` | 无 | Bool | `false` | 关闭 CRC64 校验。 |
| `--disable-checksum` | 无 | Bool | `true` | 关闭整体 CRC64 校验，仅校验分片。**v1.0.7+ 默认为 `true`；v1.0.6 及以前默认为 `false`**。 |
| `--check-point` | 无 | Bool | `true` | 开启断点续传。 |
| `--skip-dir` | 无 | Bool | `false` | 跳过"目录对象"（Key 以 `/` 结尾的 0 字节对象）。 |

**上传专用参数：**

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| `--only-current-dir` | Bool | `false` | 仅上传当前目录的文件，忽略子目录。 |
| `--disable-all-symlink` | Bool | `true` | 忽略所有软链接文件和软链接目录。 |
| `--enable-symlink-dir` | Bool | `false` | 上传软链接目录（需先将 `--disable-all-symlink` 设为 `false`）。 |
| `--acl` | String | 空 | 上传对象时设置 ACL。 |
| `--grant-read` / `--grant-read-acp` / `--grant-write-acp` / `--grant-full-control` | String | 空 | 授权指定账号访问对象。 |
| `--tags` | String | 空 | 对象标签，最多 10 个，格式 `Key1=Value1&Key2=Value2`（Key/Value 需 URL 编码）。 |
| `--forbid-overwrite` | Bool | `false` | 禁止覆盖同名对象（未启用版本控制时生效）。 |
| `--encryption-type` | String | 空 | 服务端加密方式，可选 `SSE-COS` 或 `SSE-C`。 |
| `--server-side-encryption` | String | 空 | `SSE-COS` 模式下的加密算法，可选 `AES256` 或 `SM4`。 |
| `--sse-customer-algo` | String | 空 | `SSE-C` 模式下的加密算法，可选 `AES256` 或 `SM4`。 |
| `--sse-customer-key` | String | 空 | `SSE-C` 用户提供的密钥，32 字节字符串。 |
| `--sse-customer-key-md5` | String | 空 | `SSE-C` 密钥的 MD5 值。 |

**下载专用参数：**

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| `--version-id` | String | 空 | 下载指定版本的对象（桶已启用版本控制时可用）。 |

**桶内拷贝/迁移专用参数：**

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| `--move` | Bool | `false` | 启用迁移模式，复制成功后删除源对象。仅当源与目的都为 COS 路径时有效；同一对象路径下不允许 move。 |

**OFS 差异：**

- OFS 桶的上传/下载链路会按 OFS 语义进行，建议上传时使用合理的 `--part-size` 和 `--routines` 以获得最佳性能。
- OFS 桶不支持服务端加密的部分算法，使用 `--encryption-type` 前请确认桶能力。
- OFS 桶同名目录覆盖行为与 COS 桶存在差异，建议结合 `--skip-dir` 使用。

**所需权限：**

| 场景 | 权限 Action |
|---|---|
| 上传（小文件单次 Put） | `cos:HeadBucket`、`cos:GetBucket`、`cos:HeadObject`、`cos:PutObject` |
| 上传（大文件分片） | `cos:HeadBucket`、`cos:GetBucket`、`cos:HeadObject`、`cos:InitiateMultipartUpload`、`cos:UploadPart`、`cos:CompleteMultipartUpload`、`cos:ListMultipartUploads`、`cos:ListParts` |
| 上传时设置 ACL（`--acl`/`--grant-*`） | `cos:PutObjectACL`（ACL 随上传请求头下发，相当于执行了一次 PutObjectACL） |
| 上传时设置标签（`--tags`） | `cos:PutObjectTagging`（标签随上传请求头下发） |
| 下载 | `cos:HeadBucket`、`cos:GetBucket`、`cos:HeadObject`、`cos:GetObject` |
| 下载指定版本（`--version-id`） | 同上（需桶已启用版本控制） |
| 桶间拷贝（目标桶） | `cos:GetBucket`、`cos:HeadObject`、`cos:InitiateMultipartUpload`、`cos:PutObject`、`cos:CompleteMultipartUpload` |
| 桶间拷贝（源桶） | `cos:HeadBucket`、`cos:GetBucket`、`cos:HeadObject`、`cos:GetObject` |
| 桶间迁移（`--move`） | 在拷贝权限基础上额外需源桶 `cos:DeleteObject` |

**示例：**

```shell
# 简单上传单文件
./coscli cp ./example.txt cos://examplebucket/example.txt

# 递归上传目录（忽略软链接）
./coscli cp ./data cos://examplebucket/data -r

# 上传目录，只传 .log 文件（正则写法）
# 注意：上传时匹配的是本地相对路径（Linux/macOS：logs/a.log；Windows：logs\a.log）
./coscli cp ./logs cos://examplebucket/logs -r --include '\.log$'

# 上传时设置存储类型、元信息与限速
./coscli cp ./video.mp4 cos://examplebucket/video.mp4 \
    --storage-class STANDARD_IA \
    --meta "Cache-Control:max-age=3600#Content-Type:video/mp4" \
    --rate-limiting 10

# 上传开启 SSE-COS 加密
./coscli cp ./secret.txt cos://examplebucket/secret.txt \
    --encryption-type SSE-COS --server-side-encryption AES256

# 下载单文件
./coscli cp cos://examplebucket/example.txt ./example.txt

# 递归下载目录，失败重试 3 次
./coscli cp cos://examplebucket/data ./data -r --err-retry-num 3

# 下载指定版本
./coscli cp cos://examplebucket/config.yaml ./config.yaml \
    --version-id MTg0NDY3NDI1NTk5MjQ4OTA2NA

# 桶间拷贝
./coscli cp cos://src-bucket/a.txt cos://dst-bucket/a.txt

# 桶间迁移（拷贝后删除源对象）
./coscli cp cos://src-bucket/a.txt cos://dst-bucket/a.txt --move
```

**输出示例（下载）：**

```text
INFO[2026-04-22 15:28:00] Download cos://examplebucket/data to ./data start 
Succeed: Total num: 12, size: 30 MB (30.20 MB). OK num: 12(download 12 objects).

AvgSpeed: 10.07 MB/s

cost 3.000000(s)
INFO[2026-04-22 15:28:03] Download cos://examplebucket/data to ./data Succeed: Total num: 12, size: 30 MB (30.20 MB). OK num: 12(download 12 objects). 
```

**输出示例（部分失败，退出码 2）：**

```text
WARN[2026-04-22 15:28:10] Upload ./data to cos://examplebucket/data FinishWithError: Total num: 12, size: 30 MB (30.20 MB). Error num: 2. OK num: 10(upload 10 files). 
```

### 4.2 sync（增量同步）

`sync` 命令按源路径与目的路径前缀自动识别操作类型，并通过快照（`--snapshot-path`）、更新时间（`--update`）或仅新增（`--ignore-existing`）实现增量同步，同时支持镜像模式（`--delete` 删除目的端多余文件）。

| 源路径 | 目的路径 | 操作 |
|---|---|---|
| 本地路径 | `cos://...` | 同步上传 |
| `cos://...` | 本地路径 | 同步下载 |
| `cos://...` | `cos://...` | 同步拷贝 |

> **`sync` 与 `cp` 的主要区别：**
>
> - `sync` 天然面向"保持两端一致"，默认会根据源/目的对比决定是否传输，通常配合 `--recursive` 使用
> - `sync` 独有 `--snapshot-path`/`--delete`/`--backup-dir`/`--update`/`--ignore-existing`/`--ignore-empty-file` 六个参数
> - `sync` 不提供 `--move` 参数（桶内迁移场景请使用 `cp --move`）
> - `sync` 不提供 `--version-id`（`sync` 始终按桶当前视图同步）

**语法格式：**

```shell
./coscli sync <source_path> <destination_path> [flags]
```

**通用参数：**

| 参数 | 短参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|---|
| `--recursive` | `-r` | Bool | `false` | 递归处理目录。`--delete` 必须与该选项同时使用。 |
| `--include` | 无 | String | 空 | 包含规则（需与 `-r` 配合）。**支持标准正则表达式**，详见《文档约定》。 |
| `--exclude` | 无 | String | 空 | 排除规则（需与 `-r` 配合）。**支持标准正则表达式**。 |
| `--storage-class` | 无 | String | 空 | 存储类型，可选 `STANDARD`、`STANDARD_IA`、`INTELLIGENT_TIERING`、`MAZ_STANDARD`、`MAZ_STANDARD_IA`、`ARCHIVE`、`DEEP_ARCHIVE`。下载时设置会返回错误。 |
| `--rate-limiting` | 无 | Float32 | `0` | 单链接限速，单位 MB/s。推荐取值区间 `0.1-100`；`0` 表示不限速。 |
| `--part-size` | 无 | Int64 | `32` | 分片大小（MB），最大支持 `5120`。设为 `0` 时根据文件大小自适应分块。 |
| `--thread-num` | 无 | Int | `0` | 分片并发线程数。`0` 表示按文件大小自动推算；`>0` 时覆盖自动推算且忽略 `--max-thread-num`。 |
| `--max-thread-num` | 无 | Int | `32` | `--thread-num` 为 `0` 时的自动推算上限。 |
| `--routines` | 无 | Int | `3` | 文件并发数（同时处理几个文件）。 |
| `--fail-output` | 无 | Bool | `true` | 是否将失败信息输出到文件。 |
| `--fail-output-path` | 无 | String | `coscli_output` | 失败输出目录。 |
| `--process-log` | 无 | Bool | `true` | 是否开启进程日志。 |
| `--process-log-path` | 无 | String | `coscli_output` | 进程日志输出目录。 |
| `--meta` | 无 | String | 空 | 设置对象元信息，格式 `header:value#header:value`。 |
| `--retry-num` | 无 | Int | `0` | 限流重试次数，取值 `0-100`。 |
| `--err-retry-num` | 无 | Int | `5` | 失败重试次数，取值 `0-100`。 |
| `--err-retry-interval` | 无 | Int | `0` | 失败重试间隔（秒），取值 `0-10`。`0` 表示每次随机 1-10 秒。 |
| `--disable-crc64` | 无 | Bool | `false` | 关闭 CRC64 校验。 |
| `--disable-checksum` | 无 | Bool | `true` | 关闭整体 CRC64 校验，仅校验分片。**v1.0.7+ 默认为 `true`；v1.0.6 及以前默认为 `false`**。 |
| `--check-point` | 无 | Bool | `true` | 开启断点续传。 |
| `--skip-dir` | 无 | Bool | `false` | 跳过"目录对象"（Key 以 `/` 结尾的 0 字节对象）。 |

**sync 独有参数：**

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| `--snapshot-path` | String | 空 | 本地快照目录，用于加速增量同步。首次执行会记录已成功上传/下载对象的 `LastModifiedTime`，下次执行时仅传差异文件。目录需可写；请定期手动清理历史快照。 |
| `--delete` | Bool | `false` | 删除目的端存在但源端不存在的文件。**必须与 `--recursive` 同时使用**。建议先开启桶版本控制再使用该选项，以防误删。 |
| `--backup-dir` | String | 空 | 在执行 `--delete` 前，将待删除的目的端文件备份到此目录（仅下载侧生效）。 |
| `--force` | Bool | `false` | 强制执行，不弹出二次确认。 |
| `--update` | Bool | `false` | 仅在目的对象不存在、或源对象的 `LastModifiedTime` 较新时才执行传输。 |
| `--ignore-existing` | Bool | `false` | 仅在目的对象不存在时执行传输。 |
| `--ignore-empty-file` | Bool | `false` | 跳过 0 字节文件。 |

**上传专用参数：**

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| `--only-current-dir` | Bool | `false` | 仅上传当前目录的文件，忽略子目录。 |
| `--disable-all-symlink` | Bool | `true` | 忽略所有软链接文件和软链接目录。 |
| `--enable-symlink-dir` | Bool | `false` | 上传软链接目录（需先将 `--disable-all-symlink` 设为 `false`）。 |
| `--acl` | String | 空 | 上传对象时设置 ACL。 |
| `--grant-read` / `--grant-read-acp` / `--grant-write-acp` / `--grant-full-control` | String | 空 | 授权指定账号访问对象。 |
| `--tags` | String | 空 | 对象标签，最多 10 个，格式 `Key1=Value1&Key2=Value2`（Key/Value 需 URL 编码）。 |
| `--forbid-overwrite` | Bool | `false` | 禁止覆盖同名对象（未启用版本控制时生效）。 |
| `--encryption-type` | String | 空 | 服务端加密方式，可选 `SSE-COS` 或 `SSE-C`。 |
| `--server-side-encryption` | String | 空 | `SSE-COS` 模式下的加密算法，可选 `AES256` 或 `SM4`。 |
| `--sse-customer-algo` | String | 空 | `SSE-C` 模式下的加密算法，可选 `AES256` 或 `SM4`。 |
| `--sse-customer-key` | String | 空 | `SSE-C` 用户提供的密钥，32 字节字符串。 |
| `--sse-customer-key-md5` | String | 空 | `SSE-C` 密钥的 MD5 值。 |

**OFS 差异：**

- OFS 桶的同步链路会按 OFS 语义进行，建议合理配置 `--part-size` 与 `--routines` 以获得最佳性能。
- OFS 桶不支持部分服务端加密算法，使用 `--encryption-type` 前请确认桶能力。

**所需权限：**

| 场景 | 权限 Action |
|---|---|
| 同步上传 | `cos:HeadBucket`、`cos:GetBucket`、`cos:HeadObject`、`cos:InitiateMultipartUpload`、`cos:UploadPart`、`cos:CompleteMultipartUpload`、`cos:ListMultipartUploads`、`cos:ListParts` |
| 同步下载 | `cos:HeadBucket`、`cos:GetBucket`、`cos:HeadObject`、`cos:GetObject` |
| 桶间同步（目标桶） | `cos:GetBucket`、`cos:HeadObject`、`cos:InitiateMultipartUpload`、`cos:PutObject`、`cos:CompleteMultipartUpload` |
| 桶间同步（源桶） | `cos:HeadBucket`、`cos:GetBucket`、`cos:HeadObject`、`cos:GetObject` |
| 启用 `--delete` 镜像模式 | 额外需 `cos:DeleteObject`（下载方向删除本地文件无需云端权限） |
| 同步上传时设置 ACL（`--acl`/`--grant-*`） | `cos:PutObjectACL` |
| 同步上传时设置标签（`--tags`） | `cos:PutObjectTagging` |

**示例：**

```shell
# 增量同步本地目录到 COS（首次全量，后续增量）
./coscli sync ./data cos://examplebucket/data -r \
    --snapshot-path /tmp/coscli-snapshot

# 带删除的镜像同步（目的端多余文件会被删除，建议开启版本控制）
./coscli sync ./data cos://examplebucket/data -r --delete --force

# 带备份目录的镜像同步（被删除的目的端文件备份到 ./sync-backup）
./coscli sync ./data cos://examplebucket/data -r \
    --delete --backup-dir ./sync-backup --force

# 仅同步较新的文件（按 LastModifiedTime 判断）
./coscli sync cos://examplebucket/logs ./logs -r --update

# 仅同步目的端不存在的文件（不覆盖已有文件）
./coscli sync ./data cos://examplebucket/data -r --ignore-existing

# 忽略 0 字节文件
./coscli sync ./data cos://examplebucket/data -r --ignore-empty-file

# 桶间同步
./coscli sync cos://src-bucket/ cos://dst-bucket/ -r
```

**输出示例（首次全量上传）：**

```text
INFO[2026-04-22 15:29:00] Upload ./data to cos://examplebucket/data start 
Succeed: Total num: 100, size: 258 MB (258.30 MB). OK num: 100(upload 100 files).

AvgSpeed: 25.83 MB/s

cost 10.000000(s)
INFO[2026-04-22 15:29:10] Upload ./data to cos://examplebucket/data Succeed: Total num: 100, size: 258 MB (258.30 MB). OK num: 100(upload 100 files). 
```

**输出示例（再次执行，增量同步）：**

```text
INFO[2026-04-22 15:30:00] Upload ./data to cos://examplebucket/data start 
Succeed: Total num: 100, size: 258 MB (258.30 MB). OK num: 100(upload 15 files, skip 85 files), Skip size: 220 MB (219.50 MB).

AvgSpeed: 12.93 MB/s

cost 3.000000(s)
INFO[2026-04-22 15:30:03] Upload ./data to cos://examplebucket/data Succeed: Total num: 100, size: 258 MB (258.30 MB). OK num: 100(upload 15 files, skip 85 files), Skip size: 220 MB (219.50 MB). 
```

**输出示例（带 `--delete` 的镜像同步）：**

```text
INFO[2026-04-22 15:31:00] Upload ./data to cos://examplebucket/data start 
INFO[2026-04-22 15:31:05] Delete cos://examplebucket/data/old-file.txt successfully! 
INFO[2026-04-22 15:31:05] Delete cos://examplebucket/data/stale/ successfully! 
Succeed: Total num: 100, size: 258 MB (258.30 MB). OK num: 100(upload 2 files, skip 98 files), Skip size: 256 MB (256.30 MB).

AvgSpeed: 256.00 KB/s

cost 8.000000(s)
INFO[2026-04-22 15:31:08] Upload ./data to cos://examplebucket/data Succeed: Total num: 100, size: 258 MB (258.30 MB). OK num: 100(upload 2 files, skip 98 files), Skip size: 256 MB (256.30 MB). 
```

### 4.3 rm（删除对象）

**语法格式：**

```shell
./coscli rm cos://<bucket-alias>[/prefix/] [cos://<bucket-alias>[/prefix/] ...] [flags]
```

**参数说明：**

| 参数 | 短参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|---|
| `--recursive` | `-r` | Bool | `false` | 递归删除。 |
| `--force` | `-f` | Bool | `false` | 跳过二次确认。 |
| `--only-current-dir` | 无 | Bool | `false` | 仅删除当前层级文件，不递归子目录。 |
| `--retry-num` | 无 | Int | `0` | 限流重试次数，源码 help 文案描述为 `1-10`，实际未做边界校验。推荐取值 `0-10`。 |
| `--include` | 无 | String | 空 | 包含规则。**支持标准正则表达式**，详见《文档约定》。 |
| `--exclude` | 无 | String | 空 | 排除规则。**支持标准正则表达式**。 |
| `--fail-output` | 无 | Bool | `true` | 是否将失败信息输出到文件。 |
| `--fail-output-path` | 无 | String | `coscli_output` | 失败输出目录。 |
| `--all-versions` | 无 | Bool | `false` | 删除所有历史版本（需开启版本控制，且必须与 `-r` 同时使用）。 |
| `--version-id` | 无 | String | 空 | 删除指定版本，仅支持单对象删除（不能与 `-r` 同时使用）。 |

**OFS 差异：**

- OFS 桶不支持多版本，`--all-versions`/`--version-id` 对 OFS 桶无效。

**所需权限：**

| 场景 | 权限 Action |
|---|---|
| 删除单个对象 | `cos:HeadObject`、`cos:DeleteObject` |
| 递归删除（`-r`） | `cos:HeadBucket`、`cos:GetBucket`、`cos:HeadObject`、`cos:DeleteObject`、`cos:DeleteMultipleObjects` |
| 删除指定版本（`--version-id`） | `cos:DeleteObjectVersion` |
| 删除所有历史版本（`--all-versions`） | `cos:GetBucketVersioning`、`cos:GetBucketObjectVersions`、`cos:DeleteObjectVersion` |

**示例：**

```shell
# 递归删除前缀下所有对象
./coscli rm cos://examplebucket/tmp/ -r -f

# 跳过确认，删除单个对象的指定版本
./coscli rm cos://examplebucket/a.txt \
    --version-id MTg0NDY3NDI1NTk5MjQ4OTA2NA -f

# 多桶批量删除
./coscli rm cos://bucket1/a/ cos://bucket2/b/ -r -f
```

**输出示例（递归删除）：**

```text
INFO[2026-04-22 15:30:00] Start remove prefix cos://examplebucket/tmp/ 
delete object count:50
INFO[2026-04-22 15:30:02] Remove prefix cos://examplebucket/tmp/ completed 
```

**输出示例（删除单个对象）：**

```text
INFO[2026-04-22 15:30:05] Start Delete object cos://examplebucket/a.txt 
INFO[2026-04-22 15:30:05] Are you sure you want to Delete object cos://examplebucket/a.txt? (y/n) 
y
INFO[2026-04-22 15:30:07] Delete object cos://examplebucket/a.txt successfully! 
INFO[2026-04-22 15:30:07] Delete object cos://examplebucket/a.txt Completed 
```

> **说明：** 单个对象删除在不加 `-f` 时会进入交互确认；加上 `-f` 时跳过确认步骤。

### 4.4 cat（查看对象内容）

将对象内容打印到标准输出。适合查看小文本对象。

**语法格式：**

```shell
./coscli cat cos://<bucket-alias>/<key>
```

该命令无专属参数。

**所需权限：**

| 场景 | 权限 Action |
|---|---|
| 读取对象内容 | `cos:GetObject` |

**示例：**

```shell
./coscli cat cos://examplebucket-1250000000/hello.txt
```

**输出示例：**

```text
Hello COS!
This is the content of hello.txt.
```

## 五、对象高级操作

### 5.1 hash（计算哈希值）

计算本地文件的哈希值，或展示 COS 对象的哈希值。

**语法格式：**

```shell
./coscli hash <file-path> [--type <hash-type>]
./coscli hash cos://<bucket-alias>/<key> [--type <hash-type>]
```

**参数说明：**

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| `--type` | String | `crc64` | 哈希类型，可选 `crc64` 或 `md5`。 |

> **说明：** 计算本地文件的 MD5 时，文件大小不能超过 32 MB。

**所需权限：**

| 场景 | 权限 Action |
|---|---|
| 计算本地文件哈希 | 无需权限（仅读本地文件） |
| 查看 COS 对象哈希值（CRC64 或 MD5） | `cos:HeadObject` |

> **说明：** 查看 COS 对象 CRC64 与 MD5 均从响应头（`x-cos-hash-crc64ecma` 与 `ETag`）读取，无需读取对象数据，故仅需 `cos:HeadObject` 权限。

**示例：**

```shell
# 计算本地文件 CRC64
./coscli hash ./big.bin --type crc64

# 查看 COS 对象的 MD5
./coscli hash cos://examplebucket/test.txt --type md5
```

**输出示例（本地文件 CRC64）：**

```text
INFO[2026-04-22 15:31:00] crc64-ecma:   10962195253874765732           
```

**输出示例（本地文件 MD5）：**

```text
INFO[2026-04-22 15:31:03] md5:     b1946ac92492d2347c6235b4d2611184    
INFO[2026-04-22 15:31:03] base64:  sZRqySSS0jR8YjW00mERhA==            
```

**输出示例（COS 对象 MD5）：**

```text
INFO[2026-04-22 15:31:05] md5:     e0323a9039add2978bf5b49550572c7c    
INFO[2026-04-22 15:31:05] base64:  sr7rhZP6Wv8g54u0uUne7j4=             
```

### 5.2 signurl（生成预签名 URL）

生成用于上传或下载的预签名 URL。

**语法格式：**

```shell
./coscli signurl cos://<bucket-alias>/<key> [flags]
```

**参数说明：**

| 参数 | 短参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|---|
| `--time` | `-t` | Int | `10000` | 签名有效期（秒）。 |
| `--simple-output` | 无 | Bool | `false` | 仅输出 URL 本身，不带额外日志前缀。 |
| `--method` | `-m` | String | `GET` | HTTP 方法，仅支持 `GET`（下载）与 `PUT`（上传）。 |

**所需权限：**

`signurl` 仅在本地生成签名 URL，不会调用云端接口。**但签名所用的 SecretId/SecretKey 对应的子账号，必须拥有目标操作的权限**（否则签名 URL 在实际访问时会被 COS 拒绝）：

| 方法 | 权限 Action |
|---|---|
| `GET`（下载） | `cos:GetObject` |
| `PUT`（上传） | `cos:PutObject` |

**示例：**

```shell
# 生成 1 小时有效的下载 URL
./coscli signurl cos://examplebucket/test.jpg -t 3600

# 生成可上传的预签名 URL
./coscli signurl cos://examplebucket/upload.jpg -t 600 --method PUT

# 仅输出 URL，便于脚本拼接
./coscli signurl cos://examplebucket/test.jpg -t 600 --simple-output
```

**输出示例（默认模式）：**

```text
INFO[2026-04-22 15:32:00] Signed URL:                                  
INFO[2026-04-22 15:32:00] https://examplebucket-1250000000.cos.ap-guangzhou.myqcloud.com/test.jpg?q-sign-algorithm=sha1&q-ak=AKIDxxxxxxxx&q-sign-time=1745306000%3B1745309600&q-key-time=1745306000%3B1745309600&q-header-list=host&q-url-param-list=&q-signature=abcd1234567890abcdef1234567890abcdef1234 
```

**输出示例（`--simple-output`）：**

```text
https://examplebucket-1250000000.cos.ap-guangzhou.myqcloud.com/test.jpg?q-sign-algorithm=sha1&q-ak=AKIDxxxxxxxx&q-sign-time=1745306000%3B1745306600&q-key-time=1745306000%3B1745306600&q-header-list=host&q-url-param-list=&q-signature=abcd1234567890abcdef1234567890abcdef1234
```

### 5.3 restore（恢复归档对象）

用于恢复归档（ARCHIVE）或深度归档（DEEP_ARCHIVE）存储类型的对象。

**语法格式：**

```shell
./coscli restore cos://<bucket-alias>[/prefix] [flags]
```

**参数说明：**

| 参数 | 短参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|---|
| `--recursive` | `-r` | Bool | `false` | 递归恢复前缀下所有对象。 |
| `--include` | 无 | String | 空 | 包含规则。**支持标准正则表达式**，详见《文档约定》。 |
| `--exclude` | 无 | String | 空 | 排除规则。**支持标准正则表达式**。 |
| `--days` | `-d` | Int | `3` | 临时副本有效期（天），取值 `1-365`。 |
| `--mode` | `-m` | String | `Standard` | 取回模式，可选 `Expedited`（急速）、`Standard`（标准）、`Bulk`（批量）。 |
| `--fail-output` | 无 | Bool | `true` | 是否将失败信息输出到文件。 |
| `--fail-output-path` | 无 | String | `coscli_output` | 失败输出目录。 |

**所需权限：**

| 场景 | 权限 Action |
|---|---|
| 恢复单个对象 | `cos:HeadBucket`、`cos:PostObjectRestore` |
| 递归恢复（`-r`） | `cos:HeadBucket`、`cos:GetBucket`、`cos:PostObjectRestore` |

**示例：**

```shell
# 标准模式恢复单个对象，有效期 3 天
./coscli restore cos://examplebucket/archive.zip -d 3

# 急速模式批量恢复
./coscli restore cos://examplebucket/archive/ -r -d 7 -m Expedited
```

**输出示例（批量恢复完成）：**

```text
INFO[2026-04-22 15:33:00] Start Restore examplebucket-1250000000/archive/ 
INFO[2026-04-22 15:33:05] Restore cos://examplebucket/archive/a.zip     
INFO[2026-04-22 15:33:05] Restore cos://examplebucket/archive/b.zip     
INFO[2026-04-22 15:33:10] Restore examplebucket-1250000000/archive/ completed,total num: 15,success num: 12,restore error num: 1,error type num: 2 
```

> **说明：** `error type num` 指因存储类型不支持恢复而被跳过的对象（如 STANDARD、STANDARD_IA）；`restore error num` 指实际恢复请求失败的对象。

### 5.4 symlink（软链接管理）

用于创建或查询对象软链接。

**语法格式：**

```shell
./coscli symlink --method create|get cos://<bucket-alias>/<key> --link <linkKey>
```

**参数说明：**

| 参数 | 类型 | 说明 |
|---|---|---|
| `--method` | String | **必填**，可选 `create`（创建软链接）或 `get`（查询软链接指向的对象）。 |
| `--link` | String | **必填**，软链接对象的 Key。 |

**OFS 差异：**

- OFS 桶不支持对象软链接，使用时会返回错误。

**所需权限：**

| 场景 | 权限 Action |
|---|---|
| `create`（创建软链接） | `cos:PutSymlink` |
| `get`（查询软链接指向） | `cos:GetSymlink` |

**示例：**

```shell
# 创建软链接：link.txt -> source.txt
./coscli symlink --method create cos://examplebucket/source.txt --link link.txt

# 查询软链接指向的对象
./coscli symlink --method get cos://examplebucket --link link.txt
```

**输出示例（`create`）：**

```text
INFO[2026-04-22 15:34:00] Create symlink successfully! object: source.txt, symlink: link.txt 
```

**输出示例（`get`）：**

```text
INFO[2026-04-22 15:34:05] Link-object: source.txt                      
```

### 5.5 abort（终止分片上传）

清理桶内未完成的分片上传任务，释放存储占用。

**语法格式：**

```shell
./coscli abort cos://<bucket-alias>[/prefix] [flags]
```

**参数说明：**

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| `--include` | String | 空 | 包含规则。**支持标准正则表达式**，详见《文档约定》。 |
| `--exclude` | String | 空 | 排除规则。**支持标准正则表达式**。 |
| `--fail-output` | Bool | `true` | 是否将失败信息输出到文件。 |
| `--fail-output-path` | String | `coscli_output` | 失败输出目录。 |

**所需权限：**

| 场景 | 权限 Action |
|---|---|
| 列出未完成分片任务 | `cos:ListMultipartUploads` |
| 终止分片上传任务 | `cos:AbortMultipartUpload` |

**示例：**

```shell
# 清理整个桶内的未完成分片
./coscli abort cos://examplebucket

# 清理指定前缀下的未完成分片
./coscli abort cos://examplebucket/upload/
```

**输出示例：**

```text
INFO[2026-04-22 15:35:00] Abort cos://examplebucket Start              
INFO[2026-04-22 15:35:01] Abort success! UploadID: 149373e1bd8b4c9da2bb8e0a08d77c70-xxxx,Key: big.bin 
INFO[2026-04-22 15:35:02] Abort cos://examplebucket Completed , Total: 8,8 Success, 0 Fail 
```

### 5.6 lsparts（列出分片上传任务）

列出桶内未完成的分片上传任务，或指定 uploadId 对应的分片列表。

**语法格式：**

```shell
./coscli lsparts cos://<bucket-alias>[/prefix] [flags]
```

**参数说明：**

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| `--limit` | Int | `10000` | 返回条数上限，需大于 0。 |
| `--include` | String | 空 | 包含规则。**支持标准正则表达式**，详见《文档约定》。 |
| `--exclude` | String | 空 | 排除规则。**支持标准正则表达式**。 |
| `--upload-id` | String | 空 | 指定分片上传 ID。指定后列出该任务的所有分片；不指定时列出所有未完成分片任务。 |

**所需权限：**

| 场景 | 权限 Action |
|---|---|
| 列出未完成分片任务（不带 `--upload-id`） | `cos:ListMultipartUploads` |
| 列出指定任务的分片（带 `--upload-id`） | `cos:ListParts` |

**示例：**

```shell
# 列出桶内全部未完成分片任务
./coscli lsparts cos://examplebucket

# 查看某个分片上传任务的详情
./coscli lsparts cos://examplebucket/big.bin \
    --upload-id 149373e1bd8b4c9da2bb8e0a08d77c70-xxxx
```

**输出示例（列出全部上传任务）：**

```text
        KEY        |              UPLOAD ID              |   TYPE   |       INITIATE TIME       
-------------------+-------------------------------------+----------+----------------------------
  big.bin          | 149373e1bd8b4c9da2bb8e0a08d77c70-xxxx | STANDARD | 2026-04-21T10:30:00+08:00 
  archive/data.7z  | AB12C34D56E78F90a1B2C3D4E5F6789-yyyy | ARCHIVE  | 2026-04-21T11:45:00+08:00 
-------------------+-------------------------------------+----------+----------------------------
                                                                                     TOTAL: 2     
                                                                                 ----------------
```

**输出示例（指定 `--upload-id`）：**

```text
  PARTNUMBER |                 ETAG                 |       LAST MODIFIED       |   SIZE    
-------------+--------------------------------------+---------------------------+-----------
           1 | "e0323a9039add2978bf5b49550572c7c"   | 2026-04-21T10:30:00+08:00 | 32.00 MB  
           2 | "5d41402abc4b2a76b9719d911017c592"   | 2026-04-21T10:30:30+08:00 | 32.00 MB  
           3 | "77Acba00eF3cA28e52d83fbab31A96f1"   | 2026-04-21T10:31:00+08:00 | 15.50 MB  
-------------+--------------------------------------+---------------------------+-----------
```

## 六、桶配置管理

### 6.1 bucket-acl（桶 ACL）

**语法格式：**

```shell
./coscli bucket-acl --method put|get cos://<bucket-alias> [flags]
```

**参数说明：**

| 参数 | 类型 | 说明 |
|---|---|---|
| `--method` | String | 必填，`put` 设置 ACL，`get` 查询 ACL。 |
| `--acl` | String | 桶访问权限（`put` 时使用），可选 `private`、`public-read`、`public-read-write`、`authenticated-read`。 |
| `--grant-read` | String | 授予读权限，格式 `id="100000000001",id="100000000002"`。 |
| `--grant-write` | String | 授予写权限。 |
| `--grant-read-acp` | String | 授予读 ACL 权限。 |
| `--grant-write-acp` | String | 授予写 ACL 权限。 |
| `--grant-full-control` | String | 授予完全控制权限。 |

**所需权限：**

| 场景 | 权限 Action |
|---|---|
| `put`（修改桶 ACL） | `cos:PutBucketACL` |
| `get`（查询桶 ACL） | `cos:GetBucketACL` |

**示例：**

```shell
# 查询桶 ACL
./coscli bucket-acl --method get cos://examplebucket

# 授予多个账号读权限
./coscli bucket-acl --method put cos://examplebucket \
    --grant-read "id=\"100000000003\",id=\"100000000002\""
```

**输出示例（`get`）：**

```text
  SECTION  | KEY          | VALUE                                   
-----------+--------------+-----------------------------------------
  Owner    | UIN          | 100000000001                            
+          +--------------+-----------------------------------------+
           | ID           | qcs::cam::uin/100000000001:uin/100000000001 
+          +--------------+-----------------------------------------+
           | Display Name | admin                                   
+----------+--------------+-----------------------------------------+
           |              |                                         
+----------+--------------+-----------------------------------------+
  Grant #1 | Permission   | FULL_CONTROL                            
+          +--------------+-----------------------------------------+
           | Grantee Type | CanonicalUser                           
+          +--------------+-----------------------------------------+
           | ID           | qcs::cam::uin/100000000003:uin/100000000003 
+          +--------------+-----------------------------------------+
           | Display Name | other-user                              
-----------+--------------+-----------------------------------------
Access Control List (ACL) Information

Summary:
 - Owner: admin (UIN: 100000000001)
 - Total Grants: 1
 - Permissions:
   - FULL_CONTROL: 1 grants
```

**输出示例（`put`）：**

> **说明：** `put` 操作成功后**不会打印任何日志**，退出码为 0；失败时通过标准错误输出 SDK 返回的具体错误信息。

### 6.2 bucket-encryption（桶加密）

**语法格式：**

```shell
./coscli bucket-encryption --method put|get|delete cos://<bucket-alias> [flags]
```

**参数说明：**

| 参数 | 类型 | 说明 |
|---|---|---|
| `--method` | String | 必填，可选 `put`、`get`、`delete`。 |
| `--sse-algorithm` | String | 服务端加密算法，可选 `AES256`、`SM4`、`KMS`。`AES256` 对应 SSE-COS + AES256；`SM4` 对应 SSE-COS + SM4；`KMS` 对应 SSE-KMS。 |
| `--kms-master-key-id` | String | 当 `--sse-algorithm=KMS` 时指定 KMS 主密钥（CMK），未指定则使用 COS 默认 CMK。 |
| `--kms-algorithm` | String | 当 `--sse-algorithm=KMS` 时指定 KMS 加密算法，可选 `AES256`、`SM4`，默认 `AES256`。 |

**所需权限：**

| 场景 | 权限 Action |
|---|---|
| `put`（设置加密） | `cos:PutBucketEncryption` |
| `get`（查询加密） | `cos:GetBucketEncryption` |
| `delete`（删除加密） | `cos:DeleteBucketEncryption` |
| 启用 SSE-KMS 模式 | 额外需 KMS 相关权限（如 `kms:Encrypt`、`kms:Decrypt`、`kms:GenerateDataKey`） |

**示例：**

```shell
# 开启 SSE-COS + AES256 加密
./coscli bucket-encryption --method put cos://examplebucket --sse-algorithm AES256

# 开启 SSE-KMS 加密
./coscli bucket-encryption --method put cos://examplebucket \
    --sse-algorithm KMS --kms-master-key-id kms-xxxxxxxx

# 查询桶加密配置
./coscli bucket-encryption --method get cos://examplebucket

# 关闭桶加密
./coscli bucket-encryption --method delete cos://examplebucket
```

**输出示例（`get`，已开启加密）：**

```text
    SECTION   |    KEY     |     VALUE        
--------------+------------+------------------
  Encryption  | Algorithm  | AES256           
+             +------------+-----------------+
              | KMS Key ID | Not Specified    
+             +------------+-----------------+
              | Status     | Enabled          
--------------+------------+------------------
COS Bucket Encryption Configuration

Encryption Details:
 - Type: Server-Side Encryption with COS-Managed Keys (SSE-COS)
 - Description: Tencent Cloud COS manages encryption keys
 - Security: AES-256 encryption algorithm
```

**输出示例（`get`，未配置加密）：**

> **说明：** 桶未开启服务端加密时，COS 会返回 `NoSuchEncryptionConfiguration` 错误，工具直接打印该错误并以非 0 退出码退出。

```text
ERRO[2026-04-22 15:37:00] GET https://examplebucket-1250000000.cos.ap-guangzhou.myqcloud.com/?encryption: 404 NoSuchEncryptionConfiguration(Message: The specified bucket does not have a Encryption configuration, RequestId: ..., TraceId: ...) 
```

**输出示例（`put` / `delete`）：**

> **说明：** `put` / `delete` 操作成功后**不会打印任何日志**，退出码为 0。

### 6.3 inventory（清单任务）

> **说明：** 命令注册为 `inventory`，文档中按实际使用习惯也可称为 bucket-inventory。

**语法格式：**

```shell
./coscli inventory --method put|get|list|delete|post cos://<bucket-alias> [flags]
```

**参数说明：**

| 参数 | 类型 | 说明 |
|---|---|---|
| `--method` | String | 必填，可选 `put`（创建）、`get`（查询单个）、`list`（列出所有）、`delete`（删除）、`post`（即时生成）。 |
| `--task-id` | String | 清单任务名称。有效字符：a-z、A-Z、`0-9`、`-`、`_`、`.`。`put`/`get`/`delete`/`post` 必填，`list` 不填。 |
| `--configuration` | String | 清单任务的配置内容。**支持 XML 或 JSON 两种格式**（自动识别）。**支持从本地文件读取**：当取值以 `file://` 开头时，会读取该文件的全部内容作为配置，支持绝对路径和相对路径。 |

> **从文件读取的约束：**
>
> - 文件内容会先按 JSON 解析，失败后再按 XML 解析，两者均失败返回 `unrecognized configuration format, must be JSON or XML` 错误
> - 文件不存在、路径是目录或文件为空时，均会报错并退出
> - 相对路径会根据当前工作目录解析为绝对路径

**所需权限：**

| 场景 | 权限 Action |
|---|---|
| `put`（创建清单任务） | `cos:PutBucketInventory` |
| `get`（查询单个任务） | `cos:GetBucketInventory` |
| `list`（列出全部任务） | `cos:ListBucketInventory` |
| `delete`（删除任务） | `cos:DeleteBucketInventory` |
| `post`（即时生成） | `cos:PostBucketInventory` |
| 清单写入目标桶 | 清单指定的 `Destination.Bucket` 需允许 `cos:PutObject`（需在目标桶策略中为 COS 服务授权） |

**示例：**

```shell
# 新建清单任务（配置从文件读取）
./coscli inventory --method put cos://examplebucket \
    --task-id weekly-list \
    --configuration "file:///tmp/inventory.xml"

# 查询单个清单任务
./coscli inventory --method get cos://examplebucket --task-id weekly-list

# 列出桶内所有清单任务
./coscli inventory --method list cos://examplebucket

# 删除清单任务
./coscli inventory --method delete cos://examplebucket --task-id weekly-list

# 立即触发一次清单
./coscli inventory --method post cos://examplebucket \
    --task-id adhoc-list \
    --configuration "file:///tmp/inventory.xml"
```

**输出示例（`list`，无数据）：**

```text
  ID | STATUS | SCHEDULE | INCLUDEDOBJECTVERSIONS | DESTINATION | FILTER | FIELDS  
-----+--------+----------+------------------------+-------------+--------+---------
Detailed COS Bucket Inventory Configurations

Total inventory configurations: 0
```

**输出示例（`list`，含数据）：**

```text
      ID      | STATUS  | SCHEDULE | INCLUDEDOBJECTVERSIONS |                 DESTINATION                 |     FILTER     |              FIELDS              
--------------+---------+----------+------------------------+---------------------------------------------+----------------+----------------------------------
  weekly-list | Enabled | Weekly   | Current                | qcs::cos:ap-guangzhou::logbucket-1250000000 | Prefix: logs/  | Size,LastModifiedDate,ETag       
  daily-list  | Enabled | Daily    | All                    | qcs::cos:ap-guangzhou::logbucket-1250000000 | -              | Size,StorageClass                
--------------+---------+----------+------------------------+---------------------------------------------+----------------+----------------------------------
Detailed COS Bucket Inventory Configurations

Total inventory configurations: 2
```

**输出示例（`put` / `delete` / `post`）：**

```text
INFO[2026-04-22 15:38:00] PutInventory success                         
INFO[2026-04-22 15:38:10] DeleteInventory success                      
INFO[2026-04-22 15:38:20] PostInventory success                        
```

### 6.4 bucket-policy（桶策略）

**语法格式：**

```shell
./coscli bucket-policy --method put|get|delete cos://<bucket-alias> [--policy <json>]
```

**参数说明：**

| 参数 | 类型 | 说明 |
|---|---|---|
| `--method` | String | 必填，可选 `put`、`get`、`delete`。 |
| `--policy` | String | JSON 格式的桶策略，`put` 时必填。**支持从本地文件读取**：当取值以 `file://` 开头时，会读取该文件的全部内容作为策略，支持绝对路径和相对路径。 |

> **从文件读取的约束：**
>
> - 文件格式仅支持 **JSON**（非 JSON 会返回 `unrecognized configuration format, must be JSON` 错误）
> - 文件不存在、路径是目录或文件为空时，均会报错并退出
> - 相对路径会根据当前工作目录解析为绝对路径

**所需权限：**

| 场景 | 权限 Action |
|---|---|
| `put`（设置桶策略） | `cos:PutBucketPolicy` |
| `get`（查询桶策略） | `cos:GetBucketPolicy` |
| `delete`（删除桶策略） | `cos:DeleteBucketPolicy` |

**示例：**

```shell
# 设置桶策略
./coscli bucket-policy --method put cos://examplebucket \
    --policy '{"Statement":[{"Principal":{"qcs":["qcs::cam::uin/100000000001:uin/100000000011"]},"Effect":"allow","Action":["name/cos:GetBucket"],"Resource":["qcs::cos:ap-guangzhou:uid/1250000000:examplebucket-1250000000/*"]}],"version":"2.0"}'

# 从本地文件加载策略（推荐，避免 shell 转义问题）
./coscli bucket-policy --method put cos://examplebucket \
    --policy "file:///tmp/bucket-policy.json"

# 查询桶策略
./coscli bucket-policy --method get cos://examplebucket

# 删除桶策略
./coscli bucket-policy --method delete cos://examplebucket
```

**输出示例（`get`，已配置策略）：**

```text
    SECTION    |    KEY    |                      VALUE                       
---------------+-----------+--------------------------------------------------
  Policy       | Version   | 2.0                                              
+--------------+-----------+-------------------------------------------------+
               |           |                                                  
+--------------+-----------+-------------------------------------------------+
  Statement #1 | Effect    | allow                                            
+              +-----------+-------------------------------------------------+
               | Principal | qcs:                                             
               |           |   - qcs::cam::uin/100000000001:uin/100000000011  
+              +-----------+-------------------------------------------------+
               | Action    |   - name/cos:GetBucket                           
+              +-----------+-------------------------------------------------+
               | Resource  |   - qcs::cos:ap-guangzhou:uid/1250000000:        
               |           |     examplebucket-1250000000/*                   
---------------+-----------+--------------------------------------------------
COS Bucket Policy Configuration
```

**输出示例（`get`，未配置策略）：**

```text
ERRO[2026-04-22 15:39:00] GET https://examplebucket-1250000000.cos.ap-guangzhou.myqcloud.com/?policy: 404 (Message: , RequestId: ..., TraceId: ) 
```

**输出示例（`put` / `delete`）：**

```text
INFO[2026-04-22 15:39:00] Put Bucket Policy Success                    
INFO[2026-04-22 15:39:05] Delete Bucket Policy Success                 
```

### 6.5 bucket-tagging（桶标签）

**语法格式：**

```shell
./coscli bucket-tagging --method put|add|get|delete cos://<bucket-alias> [key#value ...]
```

**参数说明：**

| 参数 | 类型 | 说明 |
|---|---|---|
| `--method` | String | 必填，可选 `put`（覆盖设置）、`add`（追加）、`get`（查询）、`delete`（删除）。 |

标签参数通过位置参数传入，格式为 `key#value`。

- `put`：至少需提供 1 个 `key#value`，会覆盖现有标签。
- `add`：追加传入的标签。
- `get`：查询全部标签，无需额外位置参数。
- `delete`：不传位置参数时删除全部标签；传入 `key#value` 时仅删除指定项。

**所需权限：**

| 场景 | 权限 Action |
|---|---|
| `put`（覆盖设置标签） | `cos:PutBucketTagging` |
| `add`（追加标签） | `cos:PutBucketTagging`、`cos:GetBucketTagging` |
| `get`（查询标签） | `cos:GetBucketTagging` |
| `delete`（删除全部标签） | `cos:DeleteBucketTagging` |
| `delete key#value`（删除指定标签） | `cos:GetBucketTagging`、`cos:PutBucketTagging` |

**示例：**

```shell
# 覆盖设置桶标签
./coscli bucket-tagging --method put cos://examplebucket tag1#test1 tag2#test2

# 追加标签
./coscli bucket-tagging --method add cos://examplebucket env#prod

# 查询桶标签
./coscli bucket-tagging --method get cos://examplebucket

# 删除指定标签
./coscli bucket-tagging --method delete cos://examplebucket tag1#test1

# 删除全部标签
./coscli bucket-tagging --method delete cos://examplebucket
```

**输出示例（`get`）：**

```text
   KEY  | VALUE   
--------+---------
   tag1 | test1   
   tag2 | test2   
    env | prod    
```

**输出示例（`put` / `add` / `delete`）：**

> **说明：** `put` / `add` / `delete` 操作成功后**不会打印任何日志**，退出码为 0。

### 6.6 bucket-versioning（桶版本控制）

**语法格式：**

```shell
./coscli bucket-versioning --method put cos://<bucket-alias> <Enabled|Suspended>
./coscli bucket-versioning --method get cos://<bucket-alias>
```

**参数说明：**

| 参数 | 类型 | 说明 |
|---|---|---|
| `--method` | String | 必填，`put` 修改版本控制状态，`get` 查询版本控制状态。 |

`put` 时第二个位置参数必填，取值 `Enabled`（启用）或 `Suspended`（暂停）。

> **说明：** `get` 返回的状态可能为 `Enabled`、`Suspended` 或 `Closed`（从未开启过版本控制的桶）。

**OFS 差异：**

- OFS 桶不支持版本控制，相关命令无效。

**所需权限：**

| 场景 | 权限 Action |
|---|---|
| `put`（修改版本控制状态） | `cos:PutBucketVersioning` |
| `get`（查询版本控制状态） | `cos:GetBucketVersioning` |

**示例：**

```shell
# 启用版本控制
./coscli bucket-versioning --method put cos://examplebucket Enabled

# 暂停版本控制
./coscli bucket-versioning --method put cos://examplebucket Suspended

# 查询版本控制状态
./coscli bucket-versioning --method get cos://examplebucket
```

**输出示例（`put`）：**

```text
INFO[2026-04-22 15:41:00] the bucket versioning status has been changed to Enabled 
```

**输出示例（`get`）：**

```text
INFO[2026-04-22 15:41:05] bucket versioning status is Enabled          
```

> **说明：** 若桶从未开启过版本控制，`get` 返回 `bucket versioning status is Closed`。

## 七、对象属性管理

### 7.1 object-acl（对象 ACL）

**语法格式：**

```shell
./coscli object-acl --method put|get cos://<bucket-alias>/<key> [flags]
```

**参数说明：**

| 参数 | 类型 | 说明 |
|---|---|---|
| `--method` | String | 必填，`put` 设置 ACL，`get` 查询 ACL。 |
| `--version-id` | String | 指定对象版本（桶已启用版本控制时可用）。 |
| `--acl` | String | 对象访问权限，可选 `default`、`private`、`public-read`。 |
| `--grant-read` / `--grant-read-acp` / `--grant-write-acp` / `--grant-full-control` | String | 授权指定账号访问对象。 |

**OFS 差异：**

- OFS 桶**不支持**对象级 ACL，使用时会返回错误。

**所需权限：**

| 场景 | 权限 Action |
|---|---|
| `put`（修改对象 ACL） | `cos:PutObjectACL` |
| `get`（查询对象 ACL） | `cos:GetObjectACL` |
| 操作指定版本（`--version-id`） | 需桶已启用版本控制，且具备对应操作的权限 |

**示例：**

```shell
# 设置对象为公有读
./coscli object-acl --method put cos://examplebucket/readme.txt --acl public-read

# 授予指定账号读权限
./coscli object-acl --method put cos://examplebucket/readme.txt \
    --grant-read "id=\"100000000003\",id=\"100000000002\""

# 查询对象 ACL
./coscli object-acl --method get cos://examplebucket/readme.txt
```

**输出示例（`get`）：**

```text
  SECTION  | KEY          | VALUE                                   
-----------+--------------+-----------------------------------------
  Owner    | UIN          | 100000000001                            
+          +--------------+-----------------------------------------+
           | ID           | qcs::cam::uin/100000000001:uin/100000000001 
+          +--------------+-----------------------------------------+
           | Display Name | admin                                   
+----------+--------------+-----------------------------------------+
           |              |                                         
+----------+--------------+-----------------------------------------+
  Grant #1 | Permission   | READ                                    
+          +--------------+-----------------------------------------+
           | Grantee Type | Group                                   
+          +--------------+-----------------------------------------+
           | URI          | http://cam.qcloud.com/groups/global/AllUsers 
-----------+--------------+-----------------------------------------
Access Control List (ACL) Information

Summary:
 - Owner: admin (UIN: 100000000001)
 - Total Grants: 1
 - Permissions:
   - READ: 1 grants
```

**输出示例（`put`）：**

> **说明：** `put` 操作成功后**不会打印任何日志**，退出码为 0。

### 7.2 object-tagging（对象标签）

**语法格式：**

```shell
./coscli object-tagging --method put|add|get|delete cos://<bucket-alias>/<key> [key#value ...] [--version-id <id>]
```

**参数说明：**

| 参数 | 类型 | 说明 |
|---|---|---|
| `--method` | String | 必填，可选 `put`、`add`、`get`、`delete`。语义与 bucket-tagging 一致。 |
| `--version-id` | String | 指定对象版本（桶已启用版本控制时可用）。 |

**OFS 差异：**

- OFS 桶**不支持**对象标签，使用时会返回错误。

**所需权限：**

| 场景 | 权限 Action |
|---|---|
| `put`（覆盖设置对象标签） | `cos:PutObjectTagging` |
| `add`（追加对象标签） | `cos:PutObjectTagging`、`cos:GetObjectTagging` |
| `get`（查询对象标签） | `cos:GetObjectTagging` |
| `delete`（删除全部对象标签） | `cos:DeleteObjectTagging` |
| `delete key#value`（删除指定对象标签） | `cos:GetObjectTagging`、`cos:PutObjectTagging` |
| 操作指定版本（`--version-id`） | 需桶已启用版本控制，且具备对应操作的权限 |

**示例：**

```shell
# 覆盖设置对象标签
./coscli object-tagging --method put cos://examplebucket/file.log tag1#test1 tag2#test2

# 追加对象标签
./coscli object-tagging --method add cos://examplebucket/file.log env#prod

# 查询对象标签（指定版本）
./coscli object-tagging --method get cos://examplebucket/file.log \
    --version-id MTg0NDY3NDI1NTk5MjQ4OTA2NA

# 删除全部对象标签
./coscli object-tagging --method delete cos://examplebucket/file.log
```

**输出示例（`get`）：**

```text
   KEY  | VALUE   
--------+---------
   tag1 | test1   
   tag2 | test2   
    env | prod    
```

**输出示例（`put` / `add` / `delete`）：**

> **说明：** `put` / `add` / `delete` 操作成功后**不会打印任何日志**，退出码为 0。

## 八、常见场景示例

### 8.1 大文件分片上传并限速

```shell
./coscli cp ./huge.zip cos://examplebucket/huge.zip \
    --part-size 64 --thread-num 8 --routines 5 \
    --rate-limiting 50 --err-retry-num 10 --check-point
```

### 8.2 带断点续传的批量下载

```shell
./coscli cp cos://examplebucket/release/ ./release -r \
    --routines 5 --check-point --err-retry-num 5
```

### 8.3 本地目录与 COS 镜像同步（目的端多余文件删除 + 备份）

```shell
./coscli sync ./site cos://examplebucket/site -r \
    --delete --backup-dir ./site-backup \
    --snapshot-path /tmp/site-snapshot --force
```

### 8.4 生成下载 URL 并分享

```shell
./coscli signurl cos://examplebucket/private.pdf -t 3600 --simple-output
```

### 8.5 归档恢复后下载

```shell
# 第 1 步：发起恢复任务
./coscli restore cos://examplebucket/archive/ -r -d 3 -m Standard

# 第 2 步：等待恢复完成后下载
./coscli cp cos://examplebucket/archive/ ./archive -r
```

### 8.6 使用临时密钥访问 COS

```shell
./coscli ls cos://examplebucket-1250000000 \
    -i <TmpSecretId> -k <TmpSecretKey> --token <SessionToken> \
    -e cos.ap-guangzhou.myqcloud.com
```

## 九、COS 与 OFS 差异汇总

| 能力 | COS 桶 | OFS 桶 |
|---|---|---|
| 多版本控制（`bucket-versioning`） | 支持 | 不支持 |
| `ls --all-versions` | 支持 | 不支持 |
| `cp --version-id` 下载指定版本 | 支持 | 不支持 |
| `rm --all-versions` / `--version-id` | 支持 | 不支持 |
| 对象软链接（`symlink`） | 支持 | 不支持 |
| 对象 ACL（`object-acl`） | 支持 | 不支持 |
| 对象标签（`object-tagging`） | 支持 | 不支持 |

> **说明：** 为避免每次请求自动探测桶类型，可在配置文件中将桶的 `ofs` 字段设为 `true`，或在命令行通过 `--bucket-type OFS` 显式指定。

## 十、配置文件参考

默认配置文件为 `$HOME/.cos.yaml`，结构如下：

```yaml
cos:
  base:
    secretid: "xxxxxxxx"
    secretkey: "xxxxxxxx"
    sessiontoken: ""
    protocol: "https"
    mode: "SecretKey"          # SecretKey 或 CvmRole
    cvmrolename: ""
    closeautoswitchhost: ""    # "true" 表示关闭备用域名切换
    disableencryption: ""      # "true" 表示禁用密钥加密存储
    disableautofetchbuckettype: ""  # "true" 表示禁用自动获取桶类型
    proxy: ""                   # 代理地址
  buckets:
    - name: "examplebucket-1250000000"
      alias: "example"
      region: "ap-guangzhou"
      endpoint: "cos.ap-guangzhou.myqcloud.com"
      ofs: false
      customized: false         # true 表示该桶使用自定义域名
```

**参数优先级：** 命令行参数 > 配置文件桶级别 > 配置文件 base 级别。

## 十一、退出码

| 退出码 | 说明 |
|---|---|
| `0` | 全部操作成功。 |
| `1` | 参数/配置错误，或命令执行过程中发生致命错误。 |
| `2` | 批量操作部分失败（如 `cp`/`sync` 中部分文件失败）。此时可检查 `fail-output-path` 下的失败记录。 |

## 十二、参考资料

- [腾讯云对象存储 COS 产品主页](https://cloud.tencent.com/product/cos)
- [coscli GitHub 仓库](https://github.com/tencentyun/coscli)
- [COS 服务端加密（SSE-COS / SSE-KMS / SSE-C）](https://cloud.tencent.com/document/product/436/18145)
- [COS 存储桶版本控制](https://cloud.tencent.com/document/product/436/19883)
- [COS 存储类型](https://cloud.tencent.com/document/product/436/33417)
