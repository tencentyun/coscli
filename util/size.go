package util

import (
	"fmt"
	"strings"
)

func FormatSize(b int64) string {
	if b < 1024 {
		return fmt.Sprintf("%d  B", b)
	} else if b < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(b)/1024)
	} else if b < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(b)/(1024*1024))
	} else if b < 1024*1024*1024*1024 {
		return fmt.Sprintf("%.2f GB", float64(b)/(1024*1024*1024))
	} else {
		return fmt.Sprintf("%.2f TB", float64(b)/(1024*1024*1024*1024))
	}
}

func formatBytes(bytes float64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes < KB:
		return fmt.Sprintf("%.2f B", bytes)
	case bytes < MB:
		return fmt.Sprintf("%.2f KB", bytes/KB)
	case bytes < GB:
		return fmt.Sprintf("%.2f MB", bytes/MB)
	case bytes < TB:
		return fmt.Sprintf("%.2f GB", bytes/GB)
	default:
		return fmt.Sprintf("%.2f TB", bytes/TB)
	}
}
func getSizeString(size int64) string {
	prefix := ""
	str := fmt.Sprintf("%d", size)
	if size < 0 {
		prefix = "-"
		str = str[1:]
	}
	len := len(str)
	strList := []string{}
	i := len % 3
	if i != 0 {
		strList = append(strList, str[0:i])
	}
	for ; i < len; i += 3 {
		strList = append(strList, str[i:i+3])
	}

	sizeStr := formatBytes(float64(size))
	return fmt.Sprintf("%s%s Byte (%s)", prefix, strings.Join(strList, ","), sizeStr)
}

func max(a, b int64) int64 {
	if a >= b {
		return a
	}
	return b
}
