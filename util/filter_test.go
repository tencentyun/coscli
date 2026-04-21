package util

import (
	"testing"
)

func TestGetFilter(t *testing.T) {
	t.Run("include 和 exclude 均为空时返回空 filters", func(t *testing.T) {
		ok, filters := GetFilter("", "")
		if !ok {
			t.Error("期望 ok=true")
		}
		if len(filters) != 0 {
			t.Errorf("期望 filters 为空，实际长度: %d", len(filters))
		}
	})

	t.Run("只有 include 时返回 1 个 filter", func(t *testing.T) {
		ok, filters := GetFilter("*.txt", "")
		if !ok {
			t.Error("期望 ok=true")
		}
		if len(filters) != 1 {
			t.Fatalf("期望 filters 长度=1，实际: %d", len(filters))
		}
		if filters[0].name != IncludePrompt {
			t.Errorf("期望 filter.name=%s，实际: %s", IncludePrompt, filters[0].name)
		}
	})

	t.Run("只有 exclude 时返回 1 个 filter", func(t *testing.T) {
		ok, filters := GetFilter("", "*.log")
		if !ok {
			t.Error("期望 ok=true")
		}
		if len(filters) != 1 {
			t.Fatalf("期望 filters 长度=1，实际: %d", len(filters))
		}
		if filters[0].name != ExcludePrompt {
			t.Errorf("期望 filter.name=%s，实际: %s", ExcludePrompt, filters[0].name)
		}
	})

	t.Run("include 和 exclude 均有时返回 2 个 filter", func(t *testing.T) {
		ok, filters := GetFilter("*.txt", "*.log")
		if !ok {
			t.Error("期望 ok=true")
		}
		if len(filters) != 2 {
			t.Fatalf("期望 filters 长度=2，实际: %d", len(filters))
		}
		if filters[0].name != IncludePrompt {
			t.Errorf("期望 filters[0].name=%s，实际: %s", IncludePrompt, filters[0].name)
		}
		if filters[1].name != ExcludePrompt {
			t.Errorf("期望 filters[1].name=%s，实际: %s", ExcludePrompt, filters[1].name)
		}
	})

	t.Run("[! 被替换为 [^", func(t *testing.T) {
		ok, filters := GetFilter("[!abc]*.txt", "")
		if !ok {
			t.Error("期望 ok=true")
		}
		if len(filters) != 1 {
			t.Fatalf("期望 filters 长度=1，实际: %d", len(filters))
		}
		if filters[0].pattern != "[^abc]*.txt" {
			t.Errorf("期望 pattern=[^abc]*.txt，实际: %s", filters[0].pattern)
		}
	})
}

func TestCosObjectMatchPatterns(t *testing.T) {
	t.Run("空 filters 时所有对象都匹配", func(t *testing.T) {
		result := cosObjectMatchPatterns("any-object.txt", []FilterOptionType{})
		if !result {
			t.Error("期望返回 true")
		}
	})

	t.Run("include 模式匹配成功", func(t *testing.T) {
		_, filters := GetFilter(".*\\.txt", "")
		result := cosObjectMatchPatterns("test.txt", filters)
		if !result {
			t.Error("期望返回 true")
		}
	})

	t.Run("include 模式不匹配", func(t *testing.T) {
		_, filters := GetFilter(".*\\.txt", "")
		result := cosObjectMatchPatterns("test.log", filters)
		if result {
			t.Error("期望返回 false")
		}
	})

	t.Run("exclude 模式排除成功", func(t *testing.T) {
		_, filters := GetFilter("", ".*\\.log")
		result := cosObjectMatchPatterns("test.log", filters)
		if result {
			t.Error("期望返回 false（被 exclude 排除）")
		}
	})

	t.Run("exclude 模式不排除", func(t *testing.T) {
		_, filters := GetFilter("", ".*\\.log")
		result := cosObjectMatchPatterns("test.txt", filters)
		if !result {
			t.Error("期望返回 true（不被 exclude 排除）")
		}
	})
}

func TestMatchFiltersForStrs(t *testing.T) {
	t.Run("空 filters 时返回所有字符串", func(t *testing.T) {
		strs := []string{"a.txt", "b.log", "c.go"}
		result := matchFiltersForStrs(strs, []FilterOptionType{})
		if len(result) != 3 {
			t.Errorf("期望 3 个结果，实际: %d", len(result))
		}
	})

	t.Run("include 过滤后只返回匹配的字符串", func(t *testing.T) {
		strs := []string{"a.txt", "b.log", "c.txt"}
		_, filters := GetFilter(".*\\.txt", "")
		result := matchFiltersForStrs(strs, filters)
		if len(result) != 2 {
			t.Errorf("期望 2 个结果，实际: %d", len(result))
		}
	})

	t.Run("exclude 过滤后排除匹配的字符串", func(t *testing.T) {
		strs := []string{"a.txt", "b.log", "c.txt"}
		_, filters := GetFilter("", ".*\\.log")
		result := matchFiltersForStrs(strs, filters)
		if len(result) != 2 {
			t.Errorf("期望 2 个结果，实际: %d", len(result))
		}
	})

	t.Run("空字符串列表返回空结果", func(t *testing.T) {
		result := matchFiltersForStrs([]string{}, []FilterOptionType{{name: IncludePrompt, pattern: ".*"}})
		if len(result) != 0 {
			t.Errorf("期望 0 个结果，实际: %d", len(result))
		}
	})
}

func TestMatchFiltersForStr(t *testing.T) {
	t.Run("空 filters 时返回 true", func(t *testing.T) {
		result := matchFiltersForStr("test.txt", []FilterOptionType{})
		if !result {
			t.Error("期望返回 true")
		}
	})

	t.Run("第一个 filter 为 include 且匹配时返回 true", func(t *testing.T) {
		filters := []FilterOptionType{{name: IncludePrompt, pattern: ".*\\.txt"}}
		result := matchFiltersForStr("test.txt", filters)
		if !result {
			t.Error("期望返回 true")
		}
	})

	t.Run("第一个 filter 为 include 且不匹配时返回 false", func(t *testing.T) {
		filters := []FilterOptionType{{name: IncludePrompt, pattern: ".*\\.txt"}}
		result := matchFiltersForStr("test.log", filters)
		if result {
			t.Error("期望返回 false")
		}
	})

	t.Run("第一个 filter 为 exclude 且匹配时返回 false", func(t *testing.T) {
		filters := []FilterOptionType{{name: ExcludePrompt, pattern: ".*\\.log"}}
		result := matchFiltersForStr("test.log", filters)
		if result {
			t.Error("期望返回 false")
		}
	})

	t.Run("多个 filter：include OR include", func(t *testing.T) {
		filters := []FilterOptionType{
			{name: IncludePrompt, pattern: ".*\\.txt"},
			{name: IncludePrompt, pattern: ".*\\.go"},
		}
		// test.go 匹配第二个 include
		result := matchFiltersForStr("test.go", filters)
		if !result {
			t.Error("期望返回 true（OR 逻辑）")
		}
	})

	t.Run("多个 filter：include AND exclude", func(t *testing.T) {
		filters := []FilterOptionType{
			{name: IncludePrompt, pattern: ".*\\.txt"},
			{name: ExcludePrompt, pattern: "temp.*"},
		}
		// test.txt 匹配 include，不匹配 exclude，应返回 true
		result := matchFiltersForStr("test.txt", filters)
		if !result {
			t.Error("期望返回 true")
		}
		// temp.txt 匹配 include，也匹配 exclude，应返回 false
		result2 := matchFiltersForStr("temp.txt", filters)
		if result2 {
			t.Error("期望返回 false（被 exclude 排除）")
		}
	})
}
