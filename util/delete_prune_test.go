package util

import (
	"testing"
)

// TestPruneParentDirsFromDelKeys 验证 sync --delete 场景下，
// 根据 srcKeys 中文件的父目录前缀，将 delKeys 中对应的本地目录条目剔除。
// 这是修复 "Windows 上 coscli sync --delete 下载后每次都提示
// file(directory) will be removed count:N" 的核心逻辑测试。
func TestPruneParentDirsFromDelKeys(t *testing.T) {
	t.Run("Windows download 场景：本地目录条目根据 COS 文件父目录被剔除", func(t *testing.T) {
		// COS 端（srcKeys）：使用 '/' 分隔，只有文件 key
		srcKeys := map[string]commonInfoType{
			"a/b/c.txt":  {key: "a/b/c.txt"},
			"a/b/d.txt":  {key: "a/b/d.txt"},
			"a/e.txt":    {key: "a/e.txt"},
			"orphan.txt": {key: "orphan.txt"},
		}
		// 本地端：Windows 下使用 '\\' 分隔，包含目录条目（以 '\\' 结尾）
		// 模拟：文件 key 已通过原差集逻辑被删除，这里只剩目录条目 + 一个独立空目录
		delKeys := map[string]commonInfoType{
			"a\\":            {key: "a\\", isDir: true},
			"a\\b\\":         {key: "a\\b\\", isDir: true},
			"unrelated\\":    {key: "unrelated\\", isDir: true}, // COS 端无对应
			"unrelated\\x\\": {key: "unrelated\\x\\", isDir: true},
		}

		pruneParentDirsFromDelKeys(srcKeys, delKeys, CpTypeDownload, false /*isLinux*/)

		// a\ 和 a\b\ 应被剔除（srcKeys 里有以它们为父目录的文件）
		if _, ok := delKeys["a\\"]; ok {
			t.Errorf("期望 a\\ 被剔除")
		}
		if _, ok := delKeys["a\\b\\"]; ok {
			t.Errorf("期望 a\\b\\ 被剔除")
		}
		// unrelated\ 和 unrelated\x\ 应保留（COS 端没有对应文件，符合 sync 一致性语义）
		if _, ok := delKeys["unrelated\\"]; !ok {
			t.Errorf("期望 unrelated\\ 保留")
		}
		if _, ok := delKeys["unrelated\\x\\"]; !ok {
			t.Errorf("期望 unrelated\\x\\ 保留")
		}
		if len(delKeys) != 2 {
			t.Errorf("期望最终 delKeys 大小为 2，实际: %d, 内容: %v", len(delKeys), delKeys)
		}
	})

	t.Run("Linux download 场景：使用 '/' 分隔符", func(t *testing.T) {
		srcKeys := map[string]commonInfoType{
			"a/b/c.txt": {key: "a/b/c.txt"},
		}
		delKeys := map[string]commonInfoType{
			"a/":         {key: "a/", isDir: true},
			"a/b/":       {key: "a/b/", isDir: true},
			"unrelated/": {key: "unrelated/", isDir: true},
		}

		pruneParentDirsFromDelKeys(srcKeys, delKeys, CpTypeDownload, true /*isLinux*/)

		if _, ok := delKeys["a/"]; ok {
			t.Errorf("期望 a/ 被剔除")
		}
		if _, ok := delKeys["a/b/"]; ok {
			t.Errorf("期望 a/b/ 被剔除")
		}
		if _, ok := delKeys["unrelated/"]; !ok {
			t.Errorf("期望 unrelated/ 保留")
		}
	})

	t.Run("CpTypeCopy 场景：始终使用 '/' 分隔符", func(t *testing.T) {
		srcKeys := map[string]commonInfoType{
			"x/y/z.txt": {key: "x/y/z.txt"},
		}
		delKeys := map[string]commonInfoType{
			"x/":   {key: "x/"},
			"x/y/": {key: "x/y/"},
			"w/":   {key: "w/"},
		}

		// CpTypeCopy 下 isLinux 不影响行为（始终用 '/')
		pruneParentDirsFromDelKeys(srcKeys, delKeys, CpTypeCopy, false)

		if _, ok := delKeys["x/"]; ok {
			t.Errorf("期望 x/ 被剔除")
		}
		if _, ok := delKeys["x/y/"]; ok {
			t.Errorf("期望 x/y/ 被剔除")
		}
		if _, ok := delKeys["w/"]; !ok {
			t.Errorf("期望 w/ 保留")
		}
	})

	t.Run("根级文件不会误删任何目录条目", func(t *testing.T) {
		srcKeys := map[string]commonInfoType{
			"root.txt": {key: "root.txt"},
		}
		delKeys := map[string]commonInfoType{
			"a/": {key: "a/"},
			"b/": {key: "b/"},
		}
		pruneParentDirsFromDelKeys(srcKeys, delKeys, CpTypeCopy, true)
		if len(delKeys) != 2 {
			t.Errorf("期望 delKeys 不变（大小=2），实际: %d", len(delKeys))
		}
	})

	t.Run("多层嵌套目录全部被剔除", func(t *testing.T) {
		srcKeys := map[string]commonInfoType{
			"a/b/c/d/e.txt": {key: "a/b/c/d/e.txt"},
		}
		delKeys := map[string]commonInfoType{
			"a\\":          {key: "a\\"},
			"a\\b\\":       {key: "a\\b\\"},
			"a\\b\\c\\":    {key: "a\\b\\c\\"},
			"a\\b\\c\\d\\": {key: "a\\b\\c\\d\\"},
		}
		pruneParentDirsFromDelKeys(srcKeys, delKeys, CpTypeDownload, false)
		if len(delKeys) != 0 {
			t.Errorf("期望所有父目录被剔除，实际剩余: %v", delKeys)
		}
	})

	t.Run("空 srcKeys 不影响 delKeys", func(t *testing.T) {
		srcKeys := map[string]commonInfoType{}
		delKeys := map[string]commonInfoType{
			"a\\": {key: "a\\"},
		}
		pruneParentDirsFromDelKeys(srcKeys, delKeys, CpTypeDownload, false)
		if len(delKeys) != 1 {
			t.Errorf("期望 delKeys 大小=1，实际: %d", len(delKeys))
		}
	})

	t.Run("COS 端有目录占位对象时也能正确剔除（key 以 / 结尾）", func(t *testing.T) {
		// COS 上有显式目录占位（少见但合法），如 "a/b/" 这种 key
		srcKeys := map[string]commonInfoType{
			"a/b/": {key: "a/b/"},
		}
		delKeys := map[string]commonInfoType{
			"a\\":    {key: "a\\"},
			"a\\b\\": {key: "a\\b\\"},
		}
		pruneParentDirsFromDelKeys(srcKeys, delKeys, CpTypeDownload, false)
		// "a/b/" 转成 "a\\b\\"，其父目录 "a\\" 被剔除
		// "a\\b\\" 本身也会被原 line 127-135 的逻辑处理（这里测的是父目录剔除，所以 a\\b\\ 是否还在不强求）
		if _, ok := delKeys["a\\"]; ok {
			t.Errorf("期望 a\\ 被剔除")
		}
	})
}
