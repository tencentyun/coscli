package util

import (
	"fmt"
)

func FormatSize(b int64) string {
	if b < 1024 {
		return fmt.Sprintf("%d  B", b)
	} else if b < 1024 * 1024 {
		return fmt.Sprintf("%.2f KB", float64(b) / 1024)
	} else if b < 1024 * 1024 * 1024 {
		return fmt.Sprintf("%.2f MB", float64(b) / (1024 * 1024))
	} else if b < 1024 * 1024 * 1024 * 1024 {
		return fmt.Sprintf("%.2f GB", float64(b) / (1024 * 1024 * 1024))
	} else {
		return fmt.Sprintf("%.2f TB", float64(b) / (1024 * 1024 * 1024 * 1024))
	}
}
