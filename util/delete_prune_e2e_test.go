package util

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestSyncDeleteWindowsScenario 真实模拟用户报告的 Windows 场景：
// "coscli sync --delete 下载文件后，每次都提示 file(directory) will be removed count:34"
// 构造 "COS 端有文件/本地已下载好" 的状态，然后执行差集 + 父目录剔除，
// 期望 delKeys 为空（没有任何目录需要删）。
func TestSyncDeleteWindowsScenario(t *testing.T) {
	// 跳过真实跑 filepath.Walk 的部分（依赖操作系统），
	// 这里直接模拟 Windows 下本地 getFileList 的输出（key 含 '\\' 分隔符，目录 key 以 '\\' 结尾）

	// 模拟 COS 端列表（用户的桶内容）：3 个目录下共 8 个文件
	srcKeys := map[string]commonInfoType{
		"dir1/a.txt":           {key: "dir1/a.txt"},
		"dir1/b.txt":           {key: "dir1/b.txt"},
		"dir1/sub1/c.txt":      {key: "dir1/sub1/c.txt"},
		"dir1/sub1/d.txt":      {key: "dir1/sub1/d.txt"},
		"dir2/e.txt":           {key: "dir2/e.txt"},
		"dir2/sub2/f.txt":      {key: "dir2/sub2/f.txt"},
		"dir3/sub3/g.txt":      {key: "dir3/sub3/g.txt"},
		"dir3/sub3/sub4/h.txt": {key: "dir3/sub3/sub4/h.txt"},
	}

	// 模拟 Windows 下本地完整下载后的状态（key 含 '\\'，目录 key 末尾带 '\\'）
	destKeys := map[string]commonInfoType{
		"dir1\\":                  {key: "dir1\\", isDir: true},
		"dir1\\sub1\\":            {key: "dir1\\sub1\\", isDir: true},
		"dir1\\a.txt":             {key: "dir1\\a.txt"},
		"dir1\\b.txt":             {key: "dir1\\b.txt"},
		"dir1\\sub1\\c.txt":       {key: "dir1\\sub1\\c.txt"},
		"dir1\\sub1\\d.txt":       {key: "dir1\\sub1\\d.txt"},
		"dir2\\":                  {key: "dir2\\", isDir: true},
		"dir2\\sub2\\":            {key: "dir2\\sub2\\", isDir: true},
		"dir2\\e.txt":             {key: "dir2\\e.txt"},
		"dir2\\sub2\\f.txt":       {key: "dir2\\sub2\\f.txt"},
		"dir3\\":                  {key: "dir3\\", isDir: true},
		"dir3\\sub3\\":            {key: "dir3\\sub3\\", isDir: true},
		"dir3\\sub3\\sub4\\":      {key: "dir3\\sub3\\sub4\\", isDir: true},
		"dir3\\sub3\\g.txt":       {key: "dir3\\sub3\\g.txt"},
		"dir3\\sub3\\sub4\\h.txt": {key: "dir3\\sub3\\sub4\\h.txt"},
	}

	// 复刻 getDeleteKeys 的核心差集逻辑
	delKeys := make(map[string]commonInfoType)
	for k, v := range destKeys {
		delKeys[k] = v
	}

	// 原有差集（模拟 Windows: isLinux=false, CpTypeDownload）
	for k := range srcKeys {
		delete(delKeys, "dir"+"\x00") // noop guard
		localKey := ""
		for _, c := range k {
			if c == '/' {
				localKey += "\\"
			} else {
				localKey += string(c)
			}
		}
		delete(delKeys, localKey)
	}

	beforePrune := len(delKeys)
	t.Logf("父目录剔除前 delKeys 大小: %d, 内容: %v", beforePrune, keysOf(delKeys))

	// 应用修复：父目录剔除
	pruneParentDirsFromDelKeys(srcKeys, delKeys, CpTypeDownload, false /*isLinux=false, Windows*/)

	afterPrune := len(delKeys)
	t.Logf("父目录剔除后 delKeys 大小: %d, 内容: %v", afterPrune, keysOf(delKeys))

	// 用户场景期望：sync 完成后本地与 COS 一致，无任何需要删除的条目
	if afterPrune != 0 {
		t.Errorf("期望 delKeys 为空（本地与 COS 完全一致），实际剩余 %d 条: %v", afterPrune, keysOf(delKeys))
	}

	// 额外断言：修复前 delKeys 确实有目录条目（即原 bug 存在）
	if beforePrune == 0 {
		t.Errorf("理论上修复前应存在目录条目噪音，但测试数据构造有误")
	}
}

// TestSyncDeleteWithUnrelatedLocalDirs 验证：
// 本地有与 COS 无关的目录条目时，修复后仍会被列入待删（符合 sync 一致性语义）
func TestSyncDeleteWithUnrelatedLocalDirs(t *testing.T) {
	srcKeys := map[string]commonInfoType{
		"dir1/a.txt": {key: "dir1/a.txt"},
	}
	delKeys := map[string]commonInfoType{
		"dir1\\":      {key: "dir1\\", isDir: true},
		"orphan\\":    {key: "orphan\\", isDir: true}, // COS 上完全没有，应被删
		"orphan\\x\\": {key: "orphan\\x\\", isDir: true},
	}

	pruneParentDirsFromDelKeys(srcKeys, delKeys, CpTypeDownload, false)

	if _, ok := delKeys["dir1\\"]; ok {
		t.Errorf("期望 dir1\\ 被剔除（COS 有对应文件）")
	}
	if _, ok := delKeys["orphan\\"]; !ok {
		t.Errorf("期望 orphan\\ 保留（COS 无对应）")
	}
	if _, ok := delKeys["orphan\\x\\"]; !ok {
		t.Errorf("期望 orphan\\x\\ 保留")
	}
}

// TestRealFilesystemSyncDelete 真实文件系统端到端测试：
// 构造本地目录 + 文件，模拟 COS 端返回同样内容 + 一个孤立空目录，
// 走完整的 pruneParentDirsFromDelKeys + DeleteLocalFiles 流程，
// 验证：本地文件/非空目录保留，孤立空目录被搬到 backup。
func TestRealFilesystemSyncDelete(t *testing.T) {
	if runtime.GOOS == "windows" {
		// 本机是 darwin/linux，仅在非 Windows 下用 '/' 模式跑一遍端到端（Windows 逻辑已由单测覆盖）
	}

	// 准备 tmp/ dest / backup 目录
	tmpRoot, err := os.MkdirTemp("", "coscli-sync-delete-e2e-")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpRoot)

	destDir := filepath.Join(tmpRoot, "dest")
	backupDir := filepath.Join(tmpRoot, "backup")
	_ = os.MkdirAll(destDir, 0755)
	_ = os.MkdirAll(backupDir, 0755)

	// 在 destDir 下创建：
	//   dir1/a.txt, dir1/sub/b.txt  (COS 上有对应文件)
	//   orphan_empty/                 (COS 上没有，空目录，应被搬 backup)
	_ = os.MkdirAll(filepath.Join(destDir, "dir1", "sub"), 0755)
	_ = os.MkdirAll(filepath.Join(destDir, "orphan_empty"), 0755)
	_ = os.WriteFile(filepath.Join(destDir, "dir1", "a.txt"), []byte("A"), 0644)
	_ = os.WriteFile(filepath.Join(destDir, "dir1", "sub", "b.txt"), []byte("B"), 0644)

	// 构造 srcKeys（COS 端），使用 '/' 分隔
	srcKeys := map[string]commonInfoType{
		"dir1/a.txt":     {key: "dir1/a.txt"},
		"dir1/sub/b.txt": {key: "dir1/sub/b.txt"},
	}

	// 构造 destKeys：用本地分隔符，模拟 getFileList 输出
	sep := string(os.PathSeparator)
	destKeys := map[string]commonInfoType{
		"dir1" + sep:                         {key: "dir1" + sep, isDir: true},
		"dir1" + sep + "sub" + sep:           {key: "dir1" + sep + "sub" + sep, isDir: true},
		"dir1" + sep + "a.txt":               {key: "dir1" + sep + "a.txt"},
		"dir1" + sep + "sub" + sep + "b.txt": {key: "dir1" + sep + "sub" + sep + "b.txt"},
		"orphan_empty" + sep:                 {key: "orphan_empty" + sep, isDir: true},
	}

	// 差集
	delKeys := make(map[string]commonInfoType)
	for k, v := range destKeys {
		delKeys[k] = v
	}
	isLinux := sep == "/"
	for k := range srcKeys {
		if isLinux {
			delete(delKeys, k)
		} else {
			// 本地 mac/linux 跑此测试时走 isLinux=true 路径，这里是 Windows 模拟（不触发）
			localK := ""
			for _, c := range k {
				if c == '/' {
					localK += sep
				} else {
					localK += string(c)
				}
			}
			delete(delKeys, localK)
		}
	}
	// 应用修复
	pruneParentDirsFromDelKeys(srcKeys, delKeys, CpTypeDownload, isLinux)

	// 验证 delKeys 只剩 orphan_empty/
	t.Logf("剔除后 delKeys: %v", keysOf(delKeys))
	if _, ok := delKeys["orphan_empty"+sep]; !ok {
		t.Fatalf("期望 orphan_empty%s 保留在 delKeys 中，实际: %v", sep, keysOf(delKeys))
	}
	for k := range delKeys {
		if k != "orphan_empty"+sep {
			t.Errorf("意外残留的 key: %s", k)
		}
	}

	// 执行真实删除流程
	fileUrl := &FileUrl{urlStr: destDir + sep}
	fo := &FileOperations{
		Operation: Operation{
			BackupDir: backupDir + sep,
			Force:     true,
		},
		CpType: CpTypeDownload,
	}
	if err := DeleteLocalFiles(delKeys, fileUrl, fo); err != nil {
		t.Fatalf("DeleteLocalFiles 失败: %v", err)
	}

	// 断言：dir1/a.txt、dir1/sub/b.txt 应保留
	if _, err := os.Stat(filepath.Join(destDir, "dir1", "a.txt")); err != nil {
		t.Errorf("期望 dir1/a.txt 保留，但已不存在: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destDir, "dir1", "sub", "b.txt")); err != nil {
		t.Errorf("期望 dir1/sub/b.txt 保留，但已不存在: %v", err)
	}
	// orphan_empty 在 dest 下应该已经不在，而在 backup 下出现
	if _, err := os.Stat(filepath.Join(destDir, "orphan_empty")); !os.IsNotExist(err) {
		t.Errorf("期望 orphan_empty 已从 dest 移走，实际 err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(backupDir, "orphan_empty")); err != nil {
		t.Errorf("期望 orphan_empty 已到达 backup，但 stat err=%v", err)
	}
}

func keysOf(m map[string]commonInfoType) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// 编译时用到 fmt 保证 import 不报错（便于后续 t.Logf 扩展）
var _ = fmt.Sprintf
