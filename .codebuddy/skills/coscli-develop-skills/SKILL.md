---
name: coscli-develop-skills
description: coscli（腾讯云 COS 命令行工具）完整开发规范，涵盖新命令开发流程、单元测试编写（goconvey + gomonkey）、桶类型判断（COS/OFS）、FileOperations 批量操作和配置命令开发等核心技能。
---

# coscli 开发技能包

## 技能描述

本技能包涵盖 coscli（腾讯云 COS 命令行工具）的完整开发规范，包括新命令开发流程、单元测试编写、桶类型判断、文件操作和配置命令开发等核心技能。

## 适用场景

- 在 coscli 项目中开发新的 CLI 命令
- 编写符合规范的单元测试
- 处理 COS/OFS 桶类型差异
- 实现文件上传、下载、复制等批量操作
- 开发配置管理相关命令

## 包含技能

| 文件 | 技能内容 |
|---|---|
| [new-command-development.md](references/new-command-development.md) | 新命令开发完整流程（命令模板、util层实现、测试编写） |
| [bucket-type-detection.md](references/bucket-type-detection.md) | 桶类型判断与 OFS/COS 分支处理 |
| [file-operations.md](references/file-operations.md) | 文件操作（上传、下载、复制、同步）的 FileOperations 使用规范 |
| [config-command.md](references/config-command.md) | 配置命令开发规范（config add/set/delete/show） |

## 快速入口

**开发新命令** → 参考 `references/new-command-development.md`，按 Step 1~7 逐步完成

**编写单测** → 参考 `references/new-command-development.md` Step 5，使用 goconvey + gomonkey，只打桩 cos SDK 方法

**处理桶类型** → 参考 `references/bucket-type-detection.md`，使用 `util.GetBucketType`

**批量文件操作** → 参考 `references/file-operations.md`，使用 `FileOperations` + `Monitor`
