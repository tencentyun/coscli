#!/bin/bash
# 腾讯云技术文档写作 SKILL - 文档质量检查脚本
# 目标环境: Linux（GNU coreutils / grep / awk）
# 用于检查生成的 Markdown 中间产物是否符合腾讯云文档规范
#
# 检查项:
#   1. 标题层级（H1 数量、H5+ 禁用）
#   2. 标题跳级（H1→H3、H2→H4 等）
#   3. 标题末尾标点（FAQ 标题允许问号）
#   4. 中英文空格（双向检查）
#   5. 常见错别字/用词
#   6. 代码块语言标注
#   7. 人称使用（符合腾讯云规范）
#   8. 标点符号（禁止感叹号）

set -e

INPUT_FILE="${1:-}"

if [ -z "$INPUT_FILE" ]; then
    echo "用法: $0 <markdown-file>"
    echo "检查 Markdown 文档是否符合腾讯云文档写作规范"
    exit 1
fi

if [ ! -f "$INPUT_FILE" ]; then
    echo "错误: 文件 '$INPUT_FILE' 不存在"
    exit 1
fi

# ============================================================
# 预处理: 剥离代码块内容，生成"纯正文"临时文件
# 标题、空格、人称等检查在纯正文上执行，避免代码块干扰
# ============================================================
TEMP_STRIPPED=$(mktemp /tmp/check-doc-stripped.XXXXXX)
trap 'rm -f "$TEMP_STRIPPED"' EXIT

awk '
    /^```/ {
        if (in_code) { in_code = 0 } else { in_code = 1 }
        next
    }
    !in_code { print }
' "$INPUT_FILE" > "$TEMP_STRIPPED"

echo "============================="
echo "腾讯云文档规范检查报告"
echo "============================="
echo "检查文件: $INPUT_FILE"
echo "检查时间: $(date '+%Y-%m-%d %H:%M:%S')"
echo "-----------------------------"
echo ""

WARNINGS=0
ERRORS=0

# ============================================================
# 检查 1: 标题层级
# ============================================================
echo "## 1. 标题层级检查"

H1_COUNT=$(grep -cP '^# (?!#)' "$TEMP_STRIPPED" || true)
if [ "$H1_COUNT" -gt 1 ]; then
    echo "  ❌ 错误: 文档包含 $H1_COUNT 个一级标题，应仅有 1 个"
    ERRORS=$((ERRORS + 1))
elif [ "$H1_COUNT" -eq 0 ]; then
    echo "  ⚠️ 警告: 文档缺少一级标题"
    WARNINGS=$((WARNINGS + 1))
else
    echo "  ✅ 通过: 一级标题数量正确"
fi

H5_COUNT=$(grep -cP '^#{5,} ' "$TEMP_STRIPPED" || true)
if [ "$H5_COUNT" -gt 0 ]; then
    echo "  ⚠️ 警告: 检测到 $H5_COUNT 个五级及以上标题，建议控制在四级以内"
    WARNINGS=$((WARNINGS + 1))
else
    echo "  ✅ 通过: 标题层级不超过四级"
fi

# ============================================================
# 检查 2: 标题跳级（H1→H3、H2→H4 等不连续层级）
# ============================================================
echo ""
echo "## 2. 标题跳级检查"
SKIP_FOUND=0
PREV_LEVEL=0

while IFS= read -r line; do
    # 提取开头的 # 号数量
    HASHES="${line%%[^#]*}"
    LEVEL=${#HASHES}
    if [ "$PREV_LEVEL" -gt 0 ] && [ "$LEVEL" -gt $((PREV_LEVEL + 1)) ]; then
        echo "  ⚠️ 警告: 标题从 H${PREV_LEVEL} 跳到 H${LEVEL}: $line"
        SKIP_FOUND=$((SKIP_FOUND + 1))
    fi
    PREV_LEVEL=$LEVEL
done < <(grep -P '^#{1,6} ' "$TEMP_STRIPPED" || true)

if [ "$SKIP_FOUND" -gt 0 ]; then
    WARNINGS=$((WARNINGS + SKIP_FOUND))
else
    echo "  ✅ 通过: 标题层级连续，无跳级"
fi

# ============================================================
# 检查 3: 标题末尾标点
# 腾讯云规范: 标题不加标点，FAQ 类标题可用问号
# ============================================================
echo ""
echo "## 3. 标题末尾标点检查"
BAD_TITLE_PUNCT=0

while IFS= read -r line; do
    # 提取标题文本（去掉 # 前缀）
    TITLE_TEXT="${line#*# }"
    # 检查末尾是否为不允许的标点（排除问号 ? ？）
    if echo "$TITLE_TEXT" | grep -Pq '[。，；：、！.,:;!]\s*$'; then
        echo "  ⚠️ 警告: 标题不应以标点结尾: $line"
        BAD_TITLE_PUNCT=$((BAD_TITLE_PUNCT + 1))
    fi
done < <(grep -P '^#{1,6} ' "$TEMP_STRIPPED" || true)

if [ "$BAD_TITLE_PUNCT" -gt 0 ]; then
    WARNINGS=$((WARNINGS + BAD_TITLE_PUNCT))
else
    echo "  ✅ 通过: 标题末尾无多余标点"
fi

# ============================================================
# 检查 4: 中英文空格（双向检查）
# 腾讯云规范: 汉字与英文之间必须有空格
# ============================================================
echo ""
echo "## 4. 中英文空格检查"
SPACE_WARN=0

# 中文后紧跟英文
MISSING_CN_EN=$(grep -Pn '[\x{4e00}-\x{9fff}][a-zA-Z]' "$TEMP_STRIPPED" | head -5 || true)
if [ -n "$MISSING_CN_EN" ]; then
    echo "  ⚠️ 警告: 发现中文后紧跟英文缺少空格:"
    while IFS= read -r line; do
        echo "    行 $line"
    done <<< "$MISSING_CN_EN"
    SPACE_WARN=$((SPACE_WARN + 1))
fi

# 英文后紧跟中文
MISSING_EN_CN=$(grep -Pn '[a-zA-Z][\x{4e00}-\x{9fff}]' "$TEMP_STRIPPED" | head -5 || true)
if [ -n "$MISSING_EN_CN" ]; then
    echo "  ⚠️ 警告: 发现英文后紧跟中文缺少空格:"
    while IFS= read -r line; do
        echo "    行 $line"
    done <<< "$MISSING_EN_CN"
    SPACE_WARN=$((SPACE_WARN + 1))
fi

if [ "$SPACE_WARN" -eq 0 ]; then
    echo "  ✅ 通过: 中英文间距检查无异常"
else
    WARNINGS=$((WARNINGS + SPACE_WARN))
fi

# ============================================================
# 检查 5: 常见错别字/用词
# ============================================================
echo ""
echo "## 5. 常见用词检查"
declare -A WORD_MAP=(
    ["登陆"]="登录"
    ["点击"]="单击"
    ["其它"]="其他"
    ["帐号"]="账号"
)
WORD_ERRORS=0

for wrong in "${!WORD_MAP[@]}"; do
    COUNT=$(grep -c "$wrong" "$TEMP_STRIPPED" || true)
    if [ "$COUNT" -gt 0 ]; then
        echo "  ⚠️ 警告: 发现 '$wrong' $COUNT 次，建议改为 '${WORD_MAP[$wrong]}'"
        WORD_ERRORS=$((WORD_ERRORS + 1))
    fi
done

if [ "$WORD_ERRORS" -eq 0 ]; then
    echo "  ✅ 通过: 常见用词检查无异常"
else
    WARNINGS=$((WARNINGS + WORD_ERRORS))
fi

# ============================================================
# 检查 6: 代码块语言标注（在原始文件上检查）
# 使用 awk 状态机精确追踪代码块开始/结束
# ============================================================
echo ""
echo "## 6. 代码块检查"

read -r TOTAL_BLOCKS UNLABELED < <(awk '
BEGIN { in_code=0; total=0; unlabeled=0 }
/^```/ {
    if (!in_code) {
        in_code = 1
        total++
        lang = $0
        sub(/^```[ \t]*/, "", lang)
        if (lang == "") unlabeled++
    } else {
        in_code = 0
    }
    next
}
END { print total, unlabeled }
' "$INPUT_FILE")

if [ "$TOTAL_BLOCKS" -eq 0 ]; then
    echo "  ✅ 通过: 文档中无代码块"
elif [ "$UNLABELED" -gt 0 ]; then
    echo "  ⚠️ 警告: 共 $TOTAL_BLOCKS 个代码块，其中 $UNLABELED 个未标注语言类型"
    WARNINGS=$((WARNINGS + 1))
else
    echo "  ✅ 通过: 全部 $TOTAL_BLOCKS 个代码块均已标注语言类型"
fi

# ============================================================
# 检查 7: 人称使用
# 腾讯云规范:
#   - 第一人称用"腾讯云"或"我们"（正确 ✅）
#   - 禁止"小编""笔者"（错误 ❌）
#   - 第二人称用"您"或"用户"（正确 ✅）
#   - 禁止"你"（非"您"部分）（警告 ⚠️）
# ============================================================
echo ""
echo "## 7. 人称使用检查"
PERSON_ERRORS=0
PERSON_WARNS=0

# 禁用的第一人称
BAD_FIRST=$(grep -Pn '小编|笔者' "$TEMP_STRIPPED" | head -3 || true)
if [ -n "$BAD_FIRST" ]; then
    echo "  ❌ 错误: 检测到禁用的第一人称（'小编'/'笔者'），应使用'腾讯云'或'我们':"
    while IFS= read -r line; do
        echo "    行 $line"
    done <<< "$BAD_FIRST"
    PERSON_ERRORS=$((PERSON_ERRORS + 1))
fi

# "你"但不是"您"：用 PCRE 负向前瞻
BAD_SECOND=$(grep -Pn '你(?!们)' "$TEMP_STRIPPED" | grep -v '您' | head -5 || true)
if [ -n "$BAD_SECOND" ]; then
    echo "  ⚠️ 警告: 检测到'你'，腾讯云规范要求使用'您':"
    while IFS= read -r line; do
        echo "    行 $line"
    done <<< "$BAD_SECOND"
    PERSON_WARNS=$((PERSON_WARNS + 1))
fi

if [ "$PERSON_ERRORS" -eq 0 ] && [ "$PERSON_WARNS" -eq 0 ]; then
    echo "  ✅ 通过: 人称使用符合规范"
fi

ERRORS=$((ERRORS + PERSON_ERRORS))
WARNINGS=$((WARNINGS + PERSON_WARNS))

# ============================================================
# 检查 8: 标点符号
# 腾讯云规范: 技术文档中不使用感叹语气
# ============================================================
echo ""
echo "## 8. 标点符号检查"
PUNCT_WARN=0

EXCLAMATION=$(grep -c '！' "$TEMP_STRIPPED" || true)
if [ "$EXCLAMATION" -gt 0 ]; then
    echo "  ⚠️ 警告: 检测到 $EXCLAMATION 个中文感叹号，技术文档中不建议使用"
    PUNCT_WARN=$((PUNCT_WARN + 1))
fi

if [ "$PUNCT_WARN" -eq 0 ]; then
    echo "  ✅ 通过: 标点符号检查无异常"
else
    WARNINGS=$((WARNINGS + PUNCT_WARN))
fi

# ============================================================
# 汇总
# ============================================================
echo ""
echo "============================="
echo "检查结果汇总"
echo "============================="
echo "  检查项数: 8"
echo "  错误数: $ERRORS"
echo "  警告数: $WARNINGS"

if [ "$ERRORS" -eq 0 ] && [ "$WARNINGS" -eq 0 ]; then
    echo "  📋 结论: 文档符合腾讯云写作规范 ✅"
    exit 0
elif [ "$ERRORS" -eq 0 ]; then
    echo "  📋 结论: 文档基本符合规范，建议修正警告项 ⚠️"
    exit 0
else
    echo "  📋 结论: 文档存在不符合规范的问题，请修正 ❌"
    exit 1
fi
