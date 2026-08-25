package util

import (
	"testing"
)

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected string
	}{
		{"0 字节", 0, "0  B"},
		{"1 字节", 1, "1  B"},
		{"1023 字节（< 1KB）", 1023, "1023  B"},
		{"1024 字节（= 1KB）", 1024, "1.00 KB"},
		{"1536 字节（1.5KB）", 1536, "1.50 KB"},
		{"1048575 字节（< 1MB）", 1048575, "1024.00 KB"},
		{"1048576 字节（= 1MB）", 1048576, "1.00 MB"},
		{"1073741823 字节（< 1GB）", 1073741823, "1024.00 MB"},
		{"1073741824 字节（= 1GB）", 1073741824, "1.00 GB"},
		{"1099511627775 字节（< 1TB）", 1099511627775, "1024.00 GB"},
		{"1099511627776 字节（= 1TB）", 1099511627776, "1.00 TB"},
		{"2199023255552 字节（2TB）", 2199023255552, "2.00 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSize(tt.input)
			if result != tt.expected {
				t.Errorf("FormatSize(%d): 期望 %q，实际 %q", tt.input, tt.expected, result)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected string
	}{
		{"小于1KB", 512, "512.00 B"},
		{"1KB", 1024, "1.00 KB"},
		{"1MB", 1024 * 1024, "1.00 MB"},
		{"1GB", 1024 * 1024 * 1024, "1.00 GB"},
		{"1TB", 1024 * 1024 * 1024 * 1024, "1.00 TB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBytes(tt.input)
			if result != tt.expected {
				t.Errorf("期望 %s，实际: %s", tt.expected, result)
			}
		})
	}
}

func TestGetSizeString(t *testing.T) {
	t.Run("正数大小格式化", func(t *testing.T) {
		result := getSizeString(1024)
		if result == "" {
			t.Error("期望 result 不为空")
		}
		if !containsStr(result, "Byte") {
			t.Errorf("期望包含 Byte，实际: %s", result)
		}
	})

	t.Run("负数大小格式化（带负号前缀）", func(t *testing.T) {
		result := getSizeString(-1024)
		if result == "" {
			t.Error("期望 result 不为空")
		}
		if result[0] != '-' {
			t.Errorf("期望以 - 开头，实际: %s", result)
		}
	})

	t.Run("大数字含千位分隔符", func(t *testing.T) {
		result := getSizeString(1000000)
		if !containsStr(result, ",") {
			t.Errorf("期望包含千位分隔符，实际: %s", result)
		}
	})

	t.Run("小于1000的数字不含千位分隔符", func(t *testing.T) {
		result := getSizeString(999)
		if result == "" {
			t.Error("期望 result 不为空")
		}
	})
}

func TestMax(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int64
		expected int64
	}{
		{"a > b", 10, 5, 10},
		{"a < b", 3, 7, 7},
		{"a == b", 5, 5, 5},
		{"负数", -1, -2, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := max(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("期望 %d，实际: %d", tt.expected, result)
			}
		})
	}
}

// containsStr 辅助函数：检查字符串是否包含子串
func containsStr(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
