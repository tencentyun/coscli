---
name: tencent-cloud-doc-writer
description: |
  在编写或管理腾讯云技术文档时应使用本 SKILL。提供两大能力：
  1. 内容生成 — 按腾讯云文档写作规范（26 项子规范 + 17 套模板）生成符合标准的 Markdown 文档，涵盖描述类、操作类、参数类（API/SDK）、故障类、FAQ 类、词汇表、公告类、联系类、协议类（SLA/服务条款）、法律法规类等文档类型。支持公有云和私有云（用户指南）场景，自动匹配模板。
  2. 系统操作 — 通过 API 查询项目、读写文档、上传图片、管理目录树，将生成的内容推送到腾讯云写写系统。
  
  当用户需要编写腾讯云文档、查询/搜索写写系统中的内容、创建或修改文档、上传图片、调整目录结构时，触发本 SKILL。
---

# 腾讯云文档写作与管理 SKILL

## 概述

提供两条执行路径：**路径 A — 写文档** 和 **路径 B — 操作文档系统**。

## 何时触发哪条路径

| 用户意图 | 执行动作 | 路径 |
|---|---|---|
| "写一篇 CVM 产品简介" | 生成文档内容 | → **路径 A** |
| "看看项目 X 下有哪些文档" | 调查询 API | → **路径 B** |
| "在项目 X 下新建一篇购买指南" | 先生成内容，再保存到系统 | → **路径 A + B** |
| "把这篇文档的标题改一下" | 调修改 API | → **路径 B** |
| "调整一下目录顺序" | 调排序 API | → **路径 B** |

快速判断：涉及"写/生成内容" → 路径 A；涉及"系统里的项目/文档/目录" → 路径 B；都涉及 → 先 A 后 B。

---

# 路径 A：写文档（内容生成）

## 目标

生成符合腾讯云写作规范的 Markdown 文档。对用户的自然语言请求，解析意图、加载匹配的规范与模板、产出 Markdown，并执行质量检查。

**输出格式始终为标准 Markdown，严禁输出 Slate JSON。**

规范体系覆盖 26 项子规范，分为四类：

| 类别 | 子规范 |
|---|---|
| 通用写作（8 项） | 标题、段落、句子、词汇、空格、标点符号、数及数量、货币符号 |
| 元素格式（10 项） | 代码、链接、举例、步骤、表格、图形、视频、界面控件、项目列表、内容引用 |
| 提示与引用（1 项） | 说明与注意 |
| 文档类型（11 项） | 描述类、操作类、参数类、故障类、常见问题、词汇表、法律法规类、产品公告、联系我们、服务等级协议、服务条款 |

Markdown 元素映射、质量检查清单、文档类型识别规则详见 `references/writing-rules-quick.md`。

---

## 工作流程

### Step 1：解析主题与类型

从用户输入中提取以下维度：

| 维度 | 说明 | 示例 |
|---|---|---|
| 产品名称 | 涉及的腾讯云产品 | CVM、COS、CLB、TKE、TDSQL |
| 文档类型 | 属于哪种文档类别 | 操作指南、产品简介、API 文档、FAQ |
| 云环境 | 公有云还是私有云 | 公有云（默认）、私有云 |
| 目标场景 | 具体功能或使用场景 | 创建实例、配置安全组 |
| 目标读者 | 文档面向的读者 | 初学者、开发者、运维、架构师 |
| 详细程度 | 期望的篇幅与深度 | 简要 / 标准 / 详细 |

**云环境识别规则：**

| 识别结果 | 触发条件 | 后续动作 |
|---|---|---|
| 私有云 | 输入包含"私有云"、"私有化部署"、"私有化"、"专有云" | 进入私有云写作流程 |
| 公有云 | 输入包含"公有云"、"公网"，或上下文明确指向公有云 | 进入公有云写作流程 |
| 未明确 | 未包含任何云环境关键词 | **主动询问用户**："请确认需要写的是**公有云**文档还是**私有云（用户指南）**文档？两者结构和规范不同。" 不要猜测，等用户明确回复后再进入 Step 2。|

**文档类型识别：** 按 `references/writing-rules-quick.md` § 5 中的优先级表匹配用户关键词。如无匹配，向用户确认后再继续。

### Step 2：加载规范

按层级加载，规范文件统一位于 `references/`。

**基础规范（始终加载）：**

| 文件 | 涵盖内容 | 搜索关键词 |
|---|---|---|
| `references/general-standard.md` | 标题、段落、句子、词汇、空格、标点、数、货币 | `## 一、标题` `## 二、段落` `## 三、句子` `## 四、词汇` `## 五、空格` `## 六、标点` `## 七、数` `## 八、货币` |
| `references/formatting.md` | 代码、链接、举例、步骤、表格、图形、视频、界面控件、项目列表、内容引用、说明与注意 | `## 一、代码` `## 二、链接` `## 三、举例` `## 四、步骤` `## 五、表格` `## 六、图形` `## 七、视频` `## 八、界面控件` `## 九、项目列表` `## 十、内容引用` `## 十一、说明与注意` |
| `references/writing-style.md` | 写作风格、目标读者、语言表达、篇幅控制、质量检查清单 | `## 一、写作风格` `## 三、语言表达` `## 四、段落` `## 五、篇幅` `## 六、质量检查` |

**按文档类型加载的专项规范：**

| 文档类型 | 文件 | 搜索关键词 |
|---|---|---|
| 描述类 | `references/doc-types.md` | `## 二、描述类` |
| 操作类 | `references/doc-types.md` | `## 三、操作类` |
| 参数类 | `references/doc-types.md` | `## 四、参数类` |
| 故障类 | `references/doc-types.md` | `## 五、故障类` |
| FAQ 类 | `references/doc-types.md` | `## 六、常见问题` |
| 词汇表类 | `references/doc-types.md` | `## 七、词汇表` |
| 法律法规类 | `references/doc-types.md` | `## 八、法律法规` |
| 公告类 | `references/doc-types.md` | `## 九、产品公告` |
| 联系类 | `references/doc-types.md` | `## 十、联系我们` |
| 协议类（SLA） | `references/doc-types.md` | `## 十一、服务等级协议` |
| 协议类（服务条款） | `references/doc-types.md` | `## 十二、服务条款` |

**私有云额外规范（Step 1 确认为私有云时加载）：**

| 私有云文档类型 | 文件 | 涵盖内容 |
|---|---|---|
| 整体结构 | `references/private-cloud-document-structure.md` | 10 大章标准架构、章节编号规则、章节间关联、关键区分 |
| 产品简介（第 1 章） | `references/private-cloud-ch1-product-intro.md` | 产品概述、相关概念、监控指标、产品关系、应用场景 |
| 快速入门 & 操作指南（第 2-3 章） | `references/private-cloud-ch2-ch3-operation.md` | 快速入门规范、操作指南结构、业务风险提示、功能特性子文档模板（含验证/回退） |
| 最佳实践 & 运维/故障（第 4-6 章） | `references/private-cloud-ch4-ch6-best-practice-troubleshoot.md` | 最佳实践案例结构、运维指南规范、故障处理五步法 |
| 常见问题 & 附录（第 7 章+附录） | `references/private-cloud-ch7-faq-appendix.md` | FAQ 格式、词汇表结构、通用操作附录、更多资源 |

**私有云规范加载策略：** 私有云规范与公有云规范互补，不替代。通用排版、标点、空格仍遵循公有云基础规范（`general-standard.md`、`formatting.md`、`writing-style.md`）；文档结构、章节体系、段落要求以私有云专项规范为准。仅加载与当前文档类型相关的章节，避免全量加载。

**模板（位于 `assets/templates/`）：**

| 文档类型 | 模板文件 | 云环境 |
|---|---|---|
| 操作指南 | `assets/templates/public-cloud/操作类文档.md` | 公有云 |
| 产品简介 | `assets/templates/public-cloud/产品简介.md` | 公有云 |
| 参数说明 | `assets/templates/public-cloud/参数说明.md` | 公有云 |
| 故障处理 | `assets/templates/public-cloud/故障处理.md` | 公有云 |
| 快速入门 | `assets/templates/public-cloud/快速入门.md` | 公有云 |
| 购买指南 | `assets/templates/public-cloud/购买指南.md` | 公有云 |
| API 文档 | `assets/templates/public-cloud/API文档.md` | 公有云 |
| SDK 文档 | `assets/templates/public-cloud/SDK文档.md` | 公有云 |
| 产品公告 | `assets/templates/public-cloud/产品公告.md` | 公有云 |
| 联系我们 | `assets/templates/public-cloud/联系我们.md` | 公有云 |
| 服务等级协议 | `assets/templates/public-cloud/服务等级协议.md` | 公有云 |
| 服务条款 | `assets/templates/public-cloud/服务条款.md` | 公有云 |
| 产品简介（用户指南） | `assets/templates/private-cloud/产品简介.md` | 私有云 |
| 快速入门（用户指南） | `assets/templates/private-cloud/快速入门.md` | 私有云 |
| 操作指南（用户指南） | `assets/templates/private-cloud/操作指南.md` | 私有云 |
| 最佳实践（用户指南） | `assets/templates/private-cloud/最佳实践.md` | 私有云 |
| 故障处理（用户指南） | `assets/templates/private-cloud/故障处理.md` | 私有云 |

**模板加载策略：** 识别文档类型后，加载对应模板文件作为骨架。私有云从 `assets/templates/private-cloud/` 加载，并加载 `references/private-cloud-document-structure.md`；公有云（默认）从 `assets/templates/public-cloud/` 加载，并加载 `references/doc-types.md`。

**术语查阅（按需加载）：** `references/terminology.md`（产品名称、通用术语、操作术语、英文术语），产品名称首次出现或术语规范化时加载。

**上下文管理：** 规范文件较大时，使用 grep 定位关键章节，避免全量加载。

### Step 3：生成内容

1. **确定结构：** 按文档类型选择结构。公有云结构见 `references/doc-types.md`；私有云见 `references/private-cloud-document-structure.md` 及各章节拆分文件。
2. **编写内容：** 严格遵循基础规范——标题/段落/句子/词汇/空格/标点/数字/货币见 `references/general-standard.md`；代码/链接/步骤/表格/图片/界面控件/列表/引用/说明注意见 `references/formatting.md`；写作风格、目标读者、篇幅见 `references/writing-style.md`。
3. **添加辅助元素：** 表格、提示框、产品全称标注、链接、示例、占位符的规则分别见 `references/formatting.md`（表格/链接/界面控件）和 `references/general-standard.md`（词汇）。

**内容筛选（基于内部资料生成外部文档时必须执行）：**

目标受众是腾讯云官网的外部用户。以"外部用户是否需要知道这个信息"为判断标准：

- ✅ 保留：功能定义与价值、使用场景、前提条件、操作步骤、参数说明、使用限制、状态说明等用户可感知的内容
- ❌ 去掉：内部接口/系统对接逻辑、审批流或工作流技术实现、内部管理后台、内部链接（Wiki/原型/TAPD）、团队间沟通结论与排期信息、未对外暴露的字段和配置项

此为默认行为。当用户明确要求保留某些内部细节时，以用户指令为准。

### Step 4：质量检查

按 `references/writing-style.md` § 六、文档质量检查清单（内容 / 语言 / 格式）自检，发现问题自动修正后输出。

可选辅助：运行 `scripts/check-doc.sh <file.md>` 执行自动化 Markdown 检查。

---

# 路径 B：操作文档系统（API 调用）

## 前置条件

- **API 地址**：`https://write.mcp.it.woa.com`（固定，无需配置）
- **Token**：从环境变量 `AI_WRITE_API_TOKEN` 读取，以 `Authorization: Bearer <token>` 传入

使用前需设置：`export AI_WRITE_API_TOKEN="你的Token"`。完整请求模板、curl 示例、响应格式详见 `references/api-reference.md`。

## 接口列表

所有 ID 字段均为字符串类型（如 `"78595548379652096"`）。完整参数表、curl 示例、错误处理详见 `references/api-reference.md`。

### 读取类

| 序号 | 接口 | 核心参数 | 用途 |
|---|---|---|---|
| 1 | `write.solution.GetCreateSolutions.v1` | `page`、`PageSize` | 我创建的项目 |
| 1b | `write.solution.GetJoinSolutions.v1` | `page`、`PageSize` | 我加入的项目 |
| 1c | `write.solution.getSolutionInfo.v1` | `solutionId` | 项目详情（返回 `versions[]`：`versionId`、`versionName`、`lang`、`isDefault`） |
| 2 | `write.node.GetTree.v1` | `solutionId`、`versionId`（可选） | 目录树 |
| 3 | `write.article.detail.v1` | `nodeId`、`contentType:"markdown"` | 返回 Markdown 格式文档内容 |

> **版本处理：** 当用户提到特定版本（如"英文版"、"v2.0"）时，先调 `getSolutionInfo` 匹配 `versionId`，后续 `GetTree` / `AddNode` / `SortNode` 等接口均带上该 `versionId`；未提版本时省略 `versionId`，系统使用默认版本。
>
> **文档搜索：** 无后端搜索接口。先调 `GetTree` 获取目录树，再本地匹配 `nodeName`。

### 写入类

| 序号 | 接口 | 核心参数 | 用途 |
|---|---|---|---|
| 4 | `write.node.AddNode.v1` | `solutionId`、`anchorType`、`nodeName`、`anchorId`、`versionId`（可选） | 创建节点。`anchorType` 取值：`top` / `up` / `down` / `sub_up` |
| 5a | `write.article.lockArticle.v1` | `nodeId` | 保存前加锁 |
| 5b | `write.article.save.v1` | `nodeId`、`content`、`contentType:"markdown"` | 保存 Markdown 内容 |
| 5c | `write.article.unLockArticle.v1` | `nodeId` | 保存后解锁 |
| 6 | `write.node.SortNode.v1` | `solutionId`、`tree` | 调整目录。需提交**完整**目录树，不是差异 |
| 7 | `write.inner.GetPresignedUploadUrl.v1` | `solutionId`、`suffix` | 获取图片预签名上传 URL。返回 `url`、`path`、`fileName` |

## 操作流程

```text
查询文档：     GetTree → 定位 nodeId → article.detail
修改文档：     GetTree → 定位 nodeId → lockArticle → article.save → unLockArticle
创建文档：     GetTree → 确定 anchorId → AddNode → lockArticle → article.save → unLockArticle
调整目录：     GetTree → 本地修改树结构 → SortNode（提交完整树）
指定版本：     getSolutionInfo → 匹配 versionId → 后续接口带上 versionId
上传图片：     GetPresignedUploadUrl → HTTP PUT 到 url → Markdown 中引用 path
```

---

# 路径 A + B：写完保存到系统

当用户要求"在某个项目下写一篇新文档"时：

```text
1. [B] GetTree → 确定插入位置（用户指定版本时先调 getSolutionInfo 获取 versionId）
2. [A] 按写作规范生成 Markdown 内容
3. [B] AddNode(带 versionId) → lockArticle → article.save(contentType:"markdown") → unLockArticle
```

---

## 注意事项

1. 用户输入不足以确定文档类型、云环境或操作目标时，主动询问澄清，不要猜测。
2. 对无法确认准确性的技术细节（API 参数、错误码），使用占位符标注并提醒用户核实。
3. 生成的文档是初稿，建议用户根据实际产品情况进行审核和调整。
4. 本 SKILL 不替代产品团队的技术审核流程。
5. **输出始终为 Markdown 格式，严禁输出 Slate JSON。**
6. 所有系统 ID 是字符串类型的大整数。
7. 保存文档前必须加锁，保存后解锁，不可跳过任一步骤。
8. `SortNode` 必须提交完整目录树，不是差异部分。
