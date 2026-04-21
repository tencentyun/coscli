package util

import (
	"testing"
)

func TestGetThreadNumByPartSize(t *testing.T) {
	tests := []struct {
		name        string
		totalSize   int64
		partSize    int64
		expectedMin int
		expectedMax int
	}{
		// partNum < 2 → threadNum = 1
		{"小文件（1 个分片）", 10 * 1024 * 1024, 100, 1, 1},
		// partNum < 4 → threadNum = 2
		{"2-3 个分片", 200 * 1024 * 1024, 100, 2, 2},
		// partNum <= 20 → threadNum = 4
		{"5-20 个分片", 500 * 1024 * 1024, 100, 4, 4},
		// partNum <= 300 → threadNum = 8
		{"21-300 个分片", 3000 * 1024 * 1024, 100, 8, 8},
		// partNum <= 500 → threadNum = 10
		{"301-500 个分片", 40000 * 1024 * 1024, 100, 10, 10},
		// partNum > 500 → threadNum = 12
		{"超过 500 个分片", 60000 * 1024 * 1024, 100, 12, 12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getThreadNumByPartSize(tt.totalSize, tt.partSize)
			if err != nil {
				t.Fatalf("期望无错误，但得到: %v", err)
			}
			if result < tt.expectedMin || result > tt.expectedMax {
				t.Errorf("getThreadNumByPartSize(%d, %d): 期望 [%d, %d]，实际 %d",
					tt.totalSize, tt.partSize, tt.expectedMin, tt.expectedMax, result)
			}
		})
	}
}
