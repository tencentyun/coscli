package util

import "fmt"

func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes < KB:
		return fmt.Sprintf("%dB", bytes)
	case bytes < MB:
		return fmt.Sprintf("%.2fKB", float64(bytes)/float64(KB))
	case bytes < GB:
		return fmt.Sprintf("%.2fMB", float64(bytes)/float64(MB))
	case bytes < TB:
		return fmt.Sprintf("%.2fGB", float64(bytes)/float64(GB))
	default:
		return fmt.Sprintf("%.2fTB", float64(bytes)/float64(TB))
	}
}
