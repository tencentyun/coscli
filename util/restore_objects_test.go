package util

import (
	"context"
	"sync"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// mockObjectPostRestoreFunc 全局 mock 变量，控制 Object.PostRestore 行为
// （Object.PostRestore 已在 TestMain 中全局打桩，被本文件及 restore_ofs_test.go / single_ops_test.go 共用）
var mockObjectPostRestoreFunc func(ctx context.Context, name string, opt *cos.ObjectRestoreOptions, id ...string) (*cos.Response, error)

// ---------------------------------------------------------------------------
// restoreCounter：并发安全 & 无跨调用状态残留
// ---------------------------------------------------------------------------

// TestRestoreCounterConcurrent 验证 restoreCounter 在高并发下计数无丢失、无数据竞争。
// 配合 go test -race 运行。
func TestRestoreCounterConcurrent(t *testing.T) {
	c := &restoreCounter{}

	const routines = 100
	const perGoroutine = 1000

	var wg sync.WaitGroup
	for i := 0; i < routines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				c.incSuccess()
				c.incFailed()
				c.incErrType()
			}
		}()
	}
	wg.Wait()

	succeed, failed, errType := c.snapshot()
	want := int64(routines * perGoroutine)
	if succeed != want || failed != want || errType != want {
		t.Fatalf("counter mismatch: succeed=%d failed=%d errType=%d, want all %d",
			succeed, failed, errType, want)
	}
}

// TestRestoreCounterFreshState 验证每次 new 出来的 counter 从 0 开始，
// 不会像原包级全局变量那样跨调用残留。
func TestRestoreCounterFreshState(t *testing.T) {
	c1 := &restoreCounter{}
	c1.incSuccess()
	c1.incSuccess()

	c2 := &restoreCounter{}
	if s, f, e := c2.snapshot(); s != 0 || f != 0 || e != 0 {
		t.Fatalf("new counter should be zero, got succeed=%d failed=%d errType=%d", s, f, e)
	}
}

// TestRestoreCounterMixedOps 验证不同计数分别累加互不干扰。
func TestRestoreCounterMixedOps(t *testing.T) {
	c := &restoreCounter{}
	c.incSuccess()
	c.incSuccess()
	c.incSuccess()
	c.incFailed()
	c.incFailed()
	c.incErrType()

	if s, f, e := c.snapshot(); s != 3 || f != 2 || e != 1 {
		t.Fatalf("unexpected counts: succeed=%d failed=%d errType=%d, want 3/2/1", s, f, e)
	}
}

// TestRestoreCounterProgress 验证进度相关计数：total/done 累加、scanEnd 标志切换，
// 以及并发下的进度快照无数据竞争（配合 go test -race）。
func TestRestoreCounterProgress(t *testing.T) {
	c := &restoreCounter{}

	// 初始应为 0 且未结束扫描
	if total, done, scanEnd := c.progressSnapshot(); total != 0 || done != 0 || scanEnd {
		t.Fatalf("fresh progress should be 0/0/false, got %d/%d/%v", total, done, scanEnd)
	}

	const n = 50
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.incTotal()
			c.incDone()
		}()
	}
	wg.Wait()

	c.setScanEnd()
	total, done, scanEnd := c.progressSnapshot()
	if total != n || done != n || !scanEnd {
		t.Fatalf("progress mismatch: total=%d done=%d scanEnd=%v, want %d/%d/true", total, done, scanEnd, n, n)
	}
}

// ---------------------------------------------------------------------------
// isRestoreType：哪些存储类型/分层可回热
// ---------------------------------------------------------------------------

func TestIsRestoreType(t *testing.T) {
	cases := []struct {
		name         string
		storageClass string
		storageTier  string
		want         bool
	}{
		{"归档", Archive, "", true},
		{"多AZ归档", MAZArchive, "", true},
		{"深度归档", DeepArchive, "", true},
		{"智能分层-归档层可回热", IntelligentTiering, StorageTierArchive, true},
		{"智能分层-深度归档层可回热", IntelligentTiering, StorageTierDeepArchive, true},
		{"多AZ智能分层-归档层可回热", MAZIntelligentTiering, StorageTierArchive, true},
		{"多AZ智能分层-深度归档层可回热", MAZIntelligentTiering, StorageTierDeepArchive, true},
		{"智能分层-频繁访问层不可回热", IntelligentTiering, "", false},
		{"标准存储不可回热", "STANDARD", "", false},
		{"低频存储不可回热", "STANDARD_IA", "", false},
		{"空存储类型不可回热", "", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			obj := cos.Object{
				StorageClass: tc.storageClass,
				StorageTier:  tc.storageTier,
			}
			if got := isRestoreType(obj); got != tc.want {
				t.Fatalf("isRestoreType(class=%q,tier=%q)=%v, want %v",
					tc.storageClass, tc.storageTier, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 过滤匹配：restore 的 include/exclude 生效逻辑
// （produceCosRestoreTasks / produceOfsRestoreTasks 均依赖 cosObjectMatchPatterns）
// ---------------------------------------------------------------------------

func TestRestoreObjectMatchFilters(t *testing.T) {
	cases := []struct {
		name    string
		include string
		exclude string
		key     string
		want    bool
	}{
		{"无过滤器全部匹配", "", "", "a/b/c.txt", true},
		{"include命中", ".*\\.txt", "", "dir/file.txt", true},
		{"include未命中", ".*\\.txt", "", "dir/file.jpg", false},
		{"exclude命中则排除", "", ".*\\.log", "dir/app.log", false},
		{"exclude未命中则保留", "", ".*\\.log", "dir/app.txt", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ok, filters := GetFilter(tc.include, tc.exclude)
			if !ok {
				t.Fatalf("GetFilter(%q,%q) returned ok=false", tc.include, tc.exclude)
			}
			if got := cosObjectMatchPatterns(tc.key, filters); got != tc.want {
				t.Fatalf("cosObjectMatchPatterns(%q) with include=%q exclude=%q = %v, want %v",
					tc.key, tc.include, tc.exclude, got, tc.want)
			}
		})
	}
}
