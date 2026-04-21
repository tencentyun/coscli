package util

import (
	"context"
	"net/http"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// resetStatisticCounters 重置所有统计全局变量，避免测试间相互影响
func resetStatisticCounters() {
	standardCnt, standardIACnt, intelligentTieringCnt, archiveCnt, deepArchiveCnt, coldCnt = 0, 0, 0, 0, 0, 0
	mazStandardCnt, mazStandardIACnt, mazIntelligentTieringCnt, mazArchiveCnt, mazColdCnt = 0, 0, 0, 0, 0
	standardSize, standardIASize, intelligentTieringSize, archiveSize, deepArchiveSize, coldSize = 0, 0, 0, 0, 0, 0
	mazStandardSize, mazStandardIASize, mazIntelligentTieringSize, mazArchiveSize, mazColdSize = 0, 0, 0, 0, 0
	deleteMarkerCnt = 0
	totalCnt = 0
	totalSize = 0
}

func TestStatisticObjects(t *testing.T) {
	tests := []struct {
		name         string
		storageClass string
		size         int64
		checkCnt     *int
		checkSize    *int64
	}{
		{"Standard", Standard, 100, &standardCnt, &standardSize},
		{"StandardIA", StandardIA, 200, &standardIACnt, &standardIASize},
		{"IntelligentTiering", IntelligentTiering, 300, &intelligentTieringCnt, &intelligentTieringSize},
		{"Archive", Archive, 400, &archiveCnt, &archiveSize},
		{"DeepArchive", DeepArchive, 500, &deepArchiveCnt, &deepArchiveSize},
		{"MAZStandard", MAZStandard, 600, &mazStandardCnt, &mazStandardSize},
		{"MAZStandardIA", MAZStandardIA, 700, &mazStandardIACnt, &mazStandardIASize},
		{"MAZIntelligentTiering", MAZIntelligentTiering, 800, &mazIntelligentTieringCnt, &mazIntelligentTieringSize},
		{"MAZArchive", MAZArchive, 900, &mazArchiveCnt, &mazArchiveSize},
		{"Cold", Cold, 1000, &coldCnt, &coldSize},
		{"MAZCold", MAZCold, 1100, &mazColdCnt, &mazColdSize},
	}

	for _, tt := range tests {
		t.Run("DU_TYPE_CATEGORIZATION_"+tt.name, func(t *testing.T) {
			resetStatisticCounters()
			obj := cos.Object{StorageClass: tt.storageClass, Size: tt.size}
			statisticObjects(obj, DU_TYPE_CATEGORIZATION)

			if *tt.checkCnt != 1 {
				t.Errorf("期望 %s cnt=1，实际: %d", tt.name, *tt.checkCnt)
			}
			if *tt.checkSize != tt.size {
				t.Errorf("期望 %s size=%d，实际: %d", tt.name, tt.size, *tt.checkSize)
			}
			if totalCnt != 1 {
				t.Errorf("期望 totalCnt=1，实际: %d", totalCnt)
			}
			if totalSize != tt.size {
				t.Errorf("期望 totalSize=%d，实际: %d", tt.size, totalSize)
			}
		})
	}

	t.Run("DU_TYPE_TOTAL 只统计总数不分类", func(t *testing.T) {
		resetStatisticCounters()
		obj := cos.Object{StorageClass: Standard, Size: 512}
		statisticObjects(obj, DU_TYPE_TOTAL)

		if standardCnt != 0 {
			t.Errorf("DU_TYPE_TOTAL 不应统计分类，standardCnt=%d", standardCnt)
		}
		if totalCnt != 1 {
			t.Errorf("期望 totalCnt=1，实际: %d", totalCnt)
		}
		if totalSize != 512 {
			t.Errorf("期望 totalSize=512，实际: %d", totalSize)
		}
	})

	t.Run("未知存储类型不影响总计", func(t *testing.T) {
		resetStatisticCounters()
		obj := cos.Object{StorageClass: "UNKNOWN_CLASS", Size: 256}
		statisticObjects(obj, DU_TYPE_CATEGORIZATION)

		if totalCnt != 1 {
			t.Errorf("期望 totalCnt=1，实际: %d", totalCnt)
		}
		if totalSize != 256 {
			t.Errorf("期望 totalSize=256，实际: %d", totalSize)
		}
	})
}

func TestStatisticObjectVersions(t *testing.T) {
	tests := []struct {
		name         string
		storageClass string
		size         int64
		checkCnt     *int
		checkSize    *int64
	}{
		{"Standard", Standard, 100, &standardCnt, &standardSize},
		{"StandardIA", StandardIA, 200, &standardIACnt, &standardIASize},
		{"IntelligentTiering", IntelligentTiering, 300, &intelligentTieringCnt, &intelligentTieringSize},
		{"Archive", Archive, 400, &archiveCnt, &archiveSize},
		{"DeepArchive", DeepArchive, 500, &deepArchiveCnt, &deepArchiveSize},
		{"MAZStandard", MAZStandard, 600, &mazStandardCnt, &mazStandardSize},
		{"MAZStandardIA", MAZStandardIA, 700, &mazStandardIACnt, &mazStandardIASize},
		{"MAZIntelligentTiering", MAZIntelligentTiering, 800, &mazIntelligentTieringCnt, &mazIntelligentTieringSize},
		{"MAZArchive", MAZArchive, 900, &mazArchiveCnt, &mazArchiveSize},
		{"Cold", Cold, 1000, &coldCnt, &coldSize},
		{"MAZCold", MAZCold, 1100, &mazColdCnt, &mazColdSize},
	}

	for _, tt := range tests {
		t.Run("DU_TYPE_CATEGORIZATION_"+tt.name, func(t *testing.T) {
			resetStatisticCounters()
			obj := cos.ListVersionsResultVersion{StorageClass: tt.storageClass, Size: tt.size}
			statisticObjectVersions(obj, DU_TYPE_CATEGORIZATION)

			if *tt.checkCnt != 1 {
				t.Errorf("期望 %s cnt=1，实际: %d", tt.name, *tt.checkCnt)
			}
			if *tt.checkSize != tt.size {
				t.Errorf("期望 %s size=%d，实际: %d", tt.name, tt.size, *tt.checkSize)
			}
			if totalCnt != 1 {
				t.Errorf("期望 totalCnt=1，实际: %d", totalCnt)
			}
			if totalSize != tt.size {
				t.Errorf("期望 totalSize=%d，实际: %d", tt.size, totalSize)
			}
		})
	}

	t.Run("DU_TYPE_TOTAL 只统计总数不分类", func(t *testing.T) {
		resetStatisticCounters()
		obj := cos.ListVersionsResultVersion{StorageClass: Standard, Size: 512}
		statisticObjectVersions(obj, DU_TYPE_TOTAL)

		if standardCnt != 0 {
			t.Errorf("DU_TYPE_TOTAL 不应统计分类，standardCnt=%d", standardCnt)
		}
		if totalCnt != 1 {
			t.Errorf("期望 totalCnt=1，实际: %d", totalCnt)
		}
		if totalSize != 512 {
			t.Errorf("期望 totalSize=512，实际: %d", totalSize)
		}
	})
}

func TestPrintStatistic(t *testing.T) {
	t.Run("allVersions=false 不输出 DeleteMarker 行", func(t *testing.T) {
		resetStatisticCounters()
		standardCnt = 5
		standardSize = 1024 * 1024
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("printStatistic panic: %v", r)
			}
		}()
		printStatistic(false)
	})

	t.Run("allVersions=true 输出 DeleteMarker 行", func(t *testing.T) {
		resetStatisticCounters()
		standardCnt = 3
		standardSize = 512 * 1024
		deleteMarkerCnt = 2
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("printStatistic panic: %v", r)
			}
		}()
		printStatistic(true)
	})

	t.Run("含 Cold 和 MAZCold 时输出对应行", func(t *testing.T) {
		resetStatisticCounters()
		coldCnt = 1
		coldSize = 100
		mazColdCnt = 2
		mazColdSize = 200
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("printStatistic panic: %v", r)
			}
		}()
		printStatistic(false)
	})
}

func TestDuObjects(t *testing.T) {
	cosUrl := &CosUrl{Bucket: "test-bucket", Object: "prefix/"}

	t.Run("COS 桶 countCosObjects 成功", func(t *testing.T) {
		resetStatisticCounters()
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents: []cos.Object{
					{Key: "prefix/file1.txt", StorageClass: Standard, Size: 1024},
					{Key: "prefix/file2.txt", StorageClass: StandardIA, Size: 2048},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := DuObjects(newTestClient(), cosUrl, nil, DU_TYPE_TOTAL, false, BucketTypeCos)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if totalCnt != 2 {
			t.Errorf("期望 totalCnt=2，实际: %d", totalCnt)
		}
	})

	t.Run("COS 桶 DU_TYPE_CATEGORIZATION 成功并打印统计", func(t *testing.T) {
		resetStatisticCounters()
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents: []cos.Object{
					{Key: "prefix/file1.txt", StorageClass: Standard, Size: 1024},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := DuObjects(newTestClient(), cosUrl, nil, DU_TYPE_CATEGORIZATION, false, BucketTypeCos)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("COS 桶 allVersions=true 时调用 countCosObjectVersions", func(t *testing.T) {
		resetStatisticCounters()
		mockBucketGetObjectVersionsFunc = func(ctx context.Context, opt *cos.BucketGetObjectVersionsOptions) (*cos.BucketGetObjectVersionsResult, *cos.Response, error) {
			return &cos.BucketGetObjectVersionsResult{
				Version: []cos.ListVersionsResultVersion{
					{Key: "prefix/file1.txt", StorageClass: Standard, Size: 1024},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := DuObjects(newTestClient(), cosUrl, nil, DU_TYPE_TOTAL, true, BucketTypeCos)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if totalCnt != 1 {
			t.Errorf("期望 totalCnt=1，实际: %d", totalCnt)
		}
	})

	t.Run("COS 桶 allVersions=true 且有 deleteMarker", func(t *testing.T) {
		resetStatisticCounters()
		mockBucketGetObjectVersionsFunc = func(ctx context.Context, opt *cos.BucketGetObjectVersionsOptions) (*cos.BucketGetObjectVersionsResult, *cos.Response, error) {
			return &cos.BucketGetObjectVersionsResult{
				DeleteMarker: []cos.ListVersionsResultDeleteMarker{
					{Key: "prefix/file1.txt"},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		err := DuObjects(newTestClient(), cosUrl, nil, DU_TYPE_CATEGORIZATION, true, BucketTypeCos)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if deleteMarkerCnt != 1 {
			t.Errorf("期望 deleteMarkerCnt=1，实际: %d", deleteMarkerCnt)
		}
	})

	t.Run("countCosObjects 过滤器生效", func(t *testing.T) {
		resetStatisticCounters()
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents: []cos.Object{
					{Key: "prefix/file1.txt", StorageClass: Standard, Size: 1024},
					{Key: "prefix/file2.log", StorageClass: Standard, Size: 2048},
					{Key: "prefix/dir/", StorageClass: Standard, Size: 0},
				},
				IsTruncated: false,
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		filters := []FilterOptionType{{name: IncludePrompt, pattern: ".*\\.txt"}}
		err := DuObjects(newTestClient(), cosUrl, filters, DU_TYPE_TOTAL, false, BucketTypeCos)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if totalCnt != 1 {
			t.Errorf("期望过滤后 totalCnt=1，实际: %d", totalCnt)
		}
	})
}
