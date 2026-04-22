//go:build symlink_verify
// +build symlink_verify

package util

// 本文件是用于一次性验证 symlink 相关问题是否真实存在的临时验证测试。
// 验证完成后可删除，或通过 `go test -tags symlink_verify ./util/ -run TestSymlinkIssue` 重跑。
//
// 验证清单：
//   P1: getFileList 对 symlink 文件 size 是否用了 lstat 值（错误）
//   P2: 统计函数 vs 传输函数 对 symlink 文件行为是否不一致
//   P3: symlink 目录环状引用是否死循环
//   P4: 嵌套 symlink 目录 object key 前缀是否错乱
//   P5: 悬空 symlink 是否导致整次遍历失败
//   P6: --only-current-dir 下 symlink 开关是否失效
//   W3: path.go UploadPathFixed 第二行覆盖第一行是否是 bug

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// collectFileListResult 调用 getFileList，收集 channel 内所有 fileInfoType
func collectFileListResult(t *testing.T, dpath string, fo *FileOperations, timeout time.Duration) ([]fileInfoType, error) {
	t.Helper()
	chFiles := make(chan fileInfoType, 1000)
	errCh := make(chan error, 1)
	go func() {
		errCh <- getFileList(dpath, chFiles, fo)
		close(chFiles)
	}()

	var files []fileInfoType
	done := make(chan struct{})
	go func() {
		for f := range chFiles {
			files = append(files, f)
		}
		close(done)
	}()

	select {
	case <-done:
		return files, <-errCh
	case <-time.After(timeout):
		return files, nil // 超时：死循环
	}
}

// ========== P1 & P2: symlink 文件 size 正确性 ==========
func TestSymlinkIssue_P1_P2_SymlinkFileSize(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip on windows")
	}
	tmpDir := filepath.Join(os.TempDir(), "coscli-sym-p1")
	os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	// 真实目标文件：100 字节
	target := filepath.Join(tmpDir, "target.bin")
	payload := strings.Repeat("x", 100)
	if err := os.WriteFile(target, []byte(payload), 0644); err != nil {
		t.Fatal(err)
	}
	// 软链接文件：link -> target.bin
	link := filepath.Join(tmpDir, "link.bin")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	// lstat 看到的 symlink 本身的 size
	li, _ := os.Lstat(link)
	lstatSize := li.Size()
	t.Logf("[背景] 软链接 lstat size=%d, 目标文件实际 size=%d", lstatSize, int64(len(payload)))

	// 子用例 A: DisableAllSymlink=false 时 getFileList 看到的 size
	t.Run("P1_getFileList_symlink文件size", func(t *testing.T) {
		fo := &FileOperations{
			Operation: Operation{DisableAllSymlink: false},
			Monitor:   &FileProcessMonitor{},
		}
		files, err := collectFileListResult(t, tmpDir+string(os.PathSeparator), fo, 5*time.Second)
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		var linkInfo *fileInfoType
		for i := range files {
			if strings.HasSuffix(files[i].filePath, "link.bin") {
				linkInfo = &files[i]
				break
			}
		}
		if linkInfo == nil {
			t.Fatal("遍历结果中没有 link.bin")
		}
		t.Logf("[结果] getFileList 对 symlink 文件报告的 size=%d", linkInfo.size)
		if linkInfo.size != 100 {
			t.Errorf("❌ P1 问题证实：getFileList 返回的 symlink 文件 size=%d，期望=100（真实目标文件大小）", linkInfo.size)
		} else {
			t.Logf("✅ P1 问题不存在：size 被正确设置为真实文件大小")
		}
	})

	// 子用例 B: getFileListStatistic 的统计行为
	t.Run("P2_statistic_symlink文件size", func(t *testing.T) {
		fo := &FileOperations{
			Operation: Operation{DisableAllSymlink: false},
			Monitor:   &FileProcessMonitor{},
		}
		if err := getFileListStatistic(tmpDir+string(os.PathSeparator), fo); err != nil {
			t.Fatalf("err=%v", err)
		}
		t.Logf("[结果] getFileListStatistic 总 size=%d（包含 target.bin 100字节 + link.bin ?字节）", fo.Monitor.TotalSize)
		// target.bin=100, 若 link.bin 也统计了真实size 则 TotalSize=200，若统计了 lstat size 则是 100+lstatSize
		if fo.Monitor.TotalSize == 200 {
			t.Logf("✅ statistic 对 symlink 文件使用真实 size")
		} else {
			t.Logf("⚠️  statistic 对 symlink 文件使用了非真实 size：TotalSize=%d", fo.Monitor.TotalSize)
		}
	})
}

// ========== P3: symlink 目录环状引用是否死循环 ==========
func TestSymlinkIssue_P3_CyclicSymlinkDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip on windows")
	}
	tmpDir := filepath.Join(os.TempDir(), "coscli-sym-p3")
	os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	// dirA/loop -> dirA  （自环）
	dirA := filepath.Join(tmpDir, "dirA")
	os.MkdirAll(dirA, 0755)
	os.WriteFile(filepath.Join(dirA, "a.txt"), []byte("a"), 0644)
	if err := os.Symlink(dirA, filepath.Join(dirA, "loop")); err != nil {
		t.Fatal(err)
	}

	fo := &FileOperations{
		Operation: Operation{DisableAllSymlink: false, EnableSymlinkDir: true},
		Monitor:   &FileProcessMonitor{},
	}

	start := time.Now()
	files, _ := collectFileListResult(t, tmpDir+string(os.PathSeparator), fo, 15*time.Second)
	elapsed := time.Since(start)
	t.Logf("[结果] 遍历耗时=%s 文件数=%d", elapsed, len(files))
	for i, f := range files {
		if i < 50 {
			t.Logf("  [%d] dir=%q filePath=%q", i, f.dir, f.filePath)
		}
	}

	if elapsed >= 15*time.Second {
		t.Errorf("❌ P3 问题证实：软链环导致遍历 15 秒未结束（死循环或超长）")
	} else if len(files) > 1000 {
		t.Errorf("❌ P3 问题证实：软链环导致文件数爆炸=%d（未做去重）", len(files))
	} else if len(files) > 10 {
		t.Errorf("⚠️  P3 疑似问题：一个真实文件+软链环，结果产出 %d 个 fileInfoType，多次重复遍历", len(files))
	} else {
		t.Logf("✅ P3 问题不存在或已被某种机制限制")
	}
}

// ========== P4: 嵌套 symlink 目录 object key 前缀是否错乱 / 多个 symlink 指向同一目录是否产生重复 key ==========
func TestSymlinkIssue_P4_NestedSymlinkDirKey(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip on windows")
	}
	tmpDir := filepath.Join(os.TempDir(), "coscli-sym-p4")
	os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	// 场景1：嵌套 symlink dir（非环）
	//   root/sub/linkA -> /tmp/.../ext/  （ext 内有 inner/b.txt, 和 linkB -> /tmp/.../ext2）
	//   /tmp/.../ext2 内有 c.txt
	ext := filepath.Join(tmpDir, "ext")
	ext2 := filepath.Join(tmpDir, "ext2")
	os.MkdirAll(filepath.Join(ext, "inner"), 0755)
	os.MkdirAll(ext2, 0755)
	os.WriteFile(filepath.Join(ext, "inner", "b.txt"), []byte("bbb"), 0644)
	os.WriteFile(filepath.Join(ext2, "c.txt"), []byte("ccc"), 0644)

	sub := filepath.Join(tmpDir, "root", "sub")
	os.MkdirAll(sub, 0755)
	os.Symlink(ext, filepath.Join(sub, "linkA"))
	os.Symlink(ext2, filepath.Join(ext, "linkB"))

	fo := &FileOperations{
		Operation: Operation{DisableAllSymlink: false, EnableSymlinkDir: true},
		Monitor:   &FileProcessMonitor{},
	}
	root := filepath.Join(tmpDir, "root") + string(os.PathSeparator)
	files, _ := collectFileListResult(t, root, fo, 5*time.Second)

	t.Logf("[场景1] 嵌套 symlink：共 %d 个 fileInfoType", len(files))
	for _, f := range files {
		t.Logf("  filePath=%q size=%d", f.filePath, f.size)
	}

	// 场景2：多个 symlink 指向**同一真实目录**，看是否导致重复 key
	tmpDir2 := filepath.Join(os.TempDir(), "coscli-sym-p4b")
	os.RemoveAll(tmpDir2)
	os.MkdirAll(tmpDir2, 0755)
	defer os.RemoveAll(tmpDir2)
	realDir := filepath.Join(tmpDir2, "real")
	os.MkdirAll(realDir, 0755)
	os.WriteFile(filepath.Join(realDir, "x.txt"), []byte("xxxxxxxxxx"), 0644) // 10 bytes
	// 用户在根下建两个 symlink 分别指向 realDir
	os.Symlink(realDir, filepath.Join(tmpDir2, "alias1"))
	os.Symlink(realDir, filepath.Join(tmpDir2, "alias2"))

	fo2 := &FileOperations{
		Operation: Operation{DisableAllSymlink: false, EnableSymlinkDir: true},
		Monitor:   &FileProcessMonitor{},
	}
	files2, _ := collectFileListResult(t, tmpDir2+string(os.PathSeparator), fo2, 5*time.Second)

	t.Logf("[场景2] 2个 symlink 指向同一真实目录：共 %d 个 fileInfoType", len(files2))
	countXTxt := 0
	for _, f := range files2 {
		t.Logf("  filePath=%q size=%d", f.filePath, f.size)
		if strings.HasSuffix(f.filePath, "x.txt") {
			countXTxt++
		}
	}
	// x.txt 的真实文件只有 1 个（在 real/x.txt），但会被通过 alias1/alias2 各列举一次
	// 如果没有做去重，COS 上会被上传 3 次：real/x.txt, alias1/x.txt, alias2/x.txt
	if countXTxt >= 3 {
		t.Errorf("❌ P4 问题证实：1 个真实文件 x.txt 被生成了 %d 个不同 COS key（通过 alias1/alias2 多路径重复），将导致 COS 上同一内容多份副本", countXTxt)
	} else if countXTxt == 2 {
		t.Logf("⚠️  P4 部分证实：x.txt 被列举 2 次")
	} else {
		t.Logf("✅ 场景2 没有产生重复 key")
	}
}

// ========== P5: 悬空 symlink 是否导致整次遍历失败 ==========
func TestSymlinkIssue_P5_DanglingSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip on windows")
	}
	tmpDir := filepath.Join(os.TempDir(), "coscli-sym-p5")
	os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, "good.txt"), []byte("good"), 0644)
	// 悬空软链（指向一个不存在的目录，模拟指向应该是dir的悬空软链）
	os.Symlink("/does/not/exist/nowhere", filepath.Join(tmpDir, "dangling"))

	// 关键：只有 EnableSymlinkDir=true 才会触发 os.Stat，进而暴露悬空问题
	fo := &FileOperations{
		Operation: Operation{DisableAllSymlink: false, EnableSymlinkDir: true},
		Monitor:   &FileProcessMonitor{},
	}
	files, err := collectFileListResult(t, tmpDir+string(os.PathSeparator), fo, 5*time.Second)
	t.Logf("[结果] err=%v 文件数=%d", err, len(files))
	if err != nil {
		t.Errorf("❌ P5 问题证实：悬空 symlink 导致 getFileList 返回 err=%v（应降级为 skip，否则一个坏链接使整批失败）", err)
	} else {
		t.Logf("✅ P5 问题不存在：遍历未因悬空 symlink 失败")
	}
}

// ========== P6: --only-current-dir 下 symlink 开关是否失效 ==========
func TestSymlinkIssue_P6_OnlyCurrentDirSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip on windows")
	}
	tmpDir := filepath.Join(os.TempDir(), "coscli-sym-p6")
	os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	// 真实文件
	os.WriteFile(filepath.Join(tmpDir, "real.txt"), []byte("real"), 0644)
	// 指向文件的软链
	os.Symlink(filepath.Join(tmpDir, "real.txt"), filepath.Join(tmpDir, "link_file"))
	// 指向目录的软链
	ext := filepath.Join(tmpDir, "ext")
	os.MkdirAll(ext, 0755)
	os.WriteFile(filepath.Join(ext, "ext.txt"), []byte("ext"), 0644)
	os.Symlink(ext, filepath.Join(tmpDir, "link_dir"))

	t.Run("DisableAllSymlink=true_期望symlink文件被跳过", func(t *testing.T) {
		fo := &FileOperations{
			Operation: Operation{DisableAllSymlink: true, OnlyCurrentDir: true},
			Monitor:   &FileProcessMonitor{},
		}
		chFiles := make(chan fileInfoType, 100)
		err := getCurrentDirFileList(tmpDir, chFiles, fo)
		close(chFiles)
		if err != nil {
			t.Fatal(err)
		}
		var keys []string
		for f := range chFiles {
			keys = append(keys, f.filePath)
		}
		t.Logf("[结果] 文件列表=%v", keys)
		// 期望：DisableAllSymlink=true 时，link_file 也应被跳过
		hasLinkFile := false
		for _, k := range keys {
			if k == "link_file" {
				hasLinkFile = true
			}
		}
		if hasLinkFile {
			t.Errorf("❌ P6 问题证实：--only-current-dir 且 DisableAllSymlink=true 时，symlink 文件 link_file 未被跳过")
		} else {
			t.Logf("✅ P6-a 问题不存在：link_file 已被跳过")
		}
	})

	t.Run("EnableSymlinkDir=true_期望link_dir被展开", func(t *testing.T) {
		fo := &FileOperations{
			Operation: Operation{DisableAllSymlink: false, EnableSymlinkDir: true, OnlyCurrentDir: true},
			Monitor:   &FileProcessMonitor{},
		}
		chFiles := make(chan fileInfoType, 100)
		err := getCurrentDirFileList(tmpDir, chFiles, fo)
		close(chFiles)
		if err != nil {
			t.Fatal(err)
		}
		var keys []string
		for f := range chFiles {
			keys = append(keys, f.filePath)
		}
		t.Logf("[结果] 文件列表=%v", keys)
		// only_current_dir 语义本身就是不递归子目录，这里不应把 link_dir 展开；
		// 只是看 link_dir 本身是否作为"目录项"被 continue 跳过（既不展开也不上传）——符合语义
		// 但若代码里 EnableSymlinkDir 完全没参与判断，则 P6 开关失效确认
		t.Logf("注：only-current-dir 下本就不递归，所以 EnableSymlinkDir 无效是语义预期，只是提示开关不生效")
	})
}

// ========== W3: UploadPathFixed path.go:50-51 bug ==========
func TestSymlinkIssue_W3_UploadPathFixedBug(t *testing.T) {
	// 第 50 行：filePath = strings.Replace(file.filePath, os.PathSeparator, "/", -1)
	// 第 51 行：filePath = strings.Replace(file.filePath, "\\", "/", -1)   ← 注意是 file.filePath 不是 filePath
	// 构造一个 filePath 既含 os.PathSeparator 又含字面反斜杠的 case，看是否两步都生效

	// macOS/Linux: os.PathSeparator = '/'
	// 构造 filePath = "a/b\\c"（含 PathSeparator '/' 和字面 '\\'）
	file := fileInfoType{filePath: "a/b\\c", dir: "/tmp"}
	_, cosPath := UploadPathFixed(file, "prefix/")
	t.Logf("[结果] UploadPathFixed 返回 cosPath=%q（期望全部分隔符替换为 /：'prefix/a/b/c'）", cosPath)
	expected := "prefix/a/b/c"
	if cosPath != expected {
		t.Errorf("❌ W3 问题证实：期望 %q，实际 %q。第51行覆盖第50行，导致替换顺序错误", expected, cosPath)
	} else {
		t.Logf("✅ W3 两种分隔符都被替换了（恰好结果等价）")
	}

	// 构造更刁钻的 case：filePath 只含 '/' 但无 '\\'：
	// 第 50 行把 '/' 替换成 '/' 等于什么都没做
	// 第 51 行把 file.filePath 的 '\\' 换成 '/'（没有 '\\'）—— 最终 filePath 是 file.filePath 原值
	// 这场景本来就无 '\\'，结果正常
	file2 := fileInfoType{filePath: "x/y/z", dir: "/tmp"}
	_, cosPath2 := UploadPathFixed(file2, "p/")
	t.Logf("[对照] 纯 '/' 路径 cosPath=%q", cosPath2)
}
