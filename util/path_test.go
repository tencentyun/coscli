package util

import (
	"os"
	"strings"
	"testing"
)

func TestIsCosPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"标准 cos:// 路径", "cos://bucket/key", true},
		{"只有 cos:// 前缀", "cos://", false},
		{"空字符串", "", false},
		{"长度不足 6", "cos:/", false},
		{"长度等于 6（cos://）", "cos://", false},
		{"本地路径", "/tmp/file.txt", false},
		{"http 路径", "http://example.com", false},
		{"相对路径", "relative/path", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCosPath(tt.input)
			if result != tt.expected {
				t.Errorf("IsCosPath(%q): 期望 %v，实际 %v", tt.input, tt.expected, result)
			}
		})
	}
}

func TestParsePath(t *testing.T) {
	t.Run("COS 路径（含 object）", func(t *testing.T) {
		bucket, path := ParsePath("cos://my-bucket/path/to/file.txt")
		if bucket != "my-bucket" {
			t.Errorf("期望 bucket=my-bucket，实际: %s", bucket)
		}
		if path != "path/to/file.txt" {
			t.Errorf("期望 path=path/to/file.txt，实际: %s", path)
		}
	})

	t.Run("COS 路径（只有 bucket）", func(t *testing.T) {
		bucket, path := ParsePath("cos://my-bucket")
		if bucket != "my-bucket" {
			t.Errorf("期望 bucket=my-bucket，实际: %s", bucket)
		}
		if path != "" {
			t.Errorf("期望 path 为空，实际: %s", path)
		}
	})

	t.Run("COS 路径（bucket 后有斜杠）", func(t *testing.T) {
		bucket, path := ParsePath("cos://my-bucket/")
		if bucket != "my-bucket" {
			t.Errorf("期望 bucket=my-bucket，实际: %s", bucket)
		}
		if path != "" {
			t.Errorf("期望 path 为空，实际: %s", path)
		}
	})

	t.Run("本地绝对路径", func(t *testing.T) {
		bucket, path := ParsePath("/tmp/local/file.txt")
		if bucket != "" {
			t.Errorf("期望 bucket 为空，实际: %s", bucket)
		}
		if path != "/tmp/local/file.txt" {
			t.Errorf("期望 path=/tmp/local/file.txt，实际: %s", path)
		}
	})

	t.Run("本地相对路径", func(t *testing.T) {
		bucket, path := ParsePath("relative/path/file.txt")
		if bucket != "" {
			t.Errorf("期望 bucket 为空，实际: %s", bucket)
		}
		if path != "relative/path/file.txt" {
			t.Errorf("期望 path=relative/path/file.txt，实际: %s", path)
		}
	})

	t.Run("~ 开头的路径展开为 home 目录", func(t *testing.T) {
		bucket, path := ParsePath("~/documents/file.txt")
		if bucket != "" {
			t.Errorf("期望 bucket 为空，实际: %s", bucket)
		}
		// ~ 应该被展开为 home 目录
		if strings.HasPrefix(path, "~") {
			t.Errorf("期望 ~ 被展开，实际 path=%s", path)
		}
		if !strings.HasSuffix(path, "/documents/file.txt") {
			t.Errorf("期望 path 以 /documents/file.txt 结尾，实际: %s", path)
		}
	})
}

func TestDownloadPathFixed(t *testing.T) {
	t.Run("目标路径以 / 结尾时追加对象相对路径", func(t *testing.T) {
		result := DownloadPathFixed("subdir/file.txt", "/tmp/download/")
		if result != "/tmp/download/subdir/file.txt" {
			t.Errorf("期望 /tmp/download/subdir/file.txt，实际: %s", result)
		}
	})

	t.Run("目标路径以 \\ 结尾时追加对象相对路径", func(t *testing.T) {
		result := DownloadPathFixed("subdir/file.txt", "C:\\download\\")
		if result != "C:\\download\\subdir/file.txt" {
			t.Errorf("期望 C:\\download\\subdir/file.txt，实际: %s", result)
		}
	})

	t.Run("目标路径不以斜杠结尾时直接返回目标路径", func(t *testing.T) {
		result := DownloadPathFixed("subdir/file.txt", "/tmp/download/output.txt")
		if result != "/tmp/download/output.txt" {
			t.Errorf("期望 /tmp/download/output.txt，实际: %s", result)
		}
	})
}

func TestCopyPathFixed(t *testing.T) {
	t.Run("目标路径为空时追加对象相对路径", func(t *testing.T) {
		result := copyPathFixed("subdir/file.txt", "")
		if result != "subdir/file.txt" {
			t.Errorf("期望 subdir/file.txt，实际: %s", result)
		}
	})

	t.Run("目标路径以 / 结尾时追加对象相对路径", func(t *testing.T) {
		result := copyPathFixed("subdir/file.txt", "dest/")
		if result != "dest/subdir/file.txt" {
			t.Errorf("期望 dest/subdir/file.txt，实际: %s", result)
		}
	})

	t.Run("目标路径不以斜杠结尾时直接返回目标路径", func(t *testing.T) {
		result := copyPathFixed("subdir/file.txt", "dest/output.txt")
		if result != "dest/output.txt" {
			t.Errorf("期望 dest/output.txt，实际: %s", result)
		}
	})
}

func TestUploadPathFixed(t *testing.T) {
	t.Run("cosPath 为空时追加文件路径", func(t *testing.T) {
		fi := fileInfoType{dir: "/tmp", filePath: "subdir/file.txt"}
		localPath, cosPath := UploadPathFixed(fi, "")
		if cosPath != "subdir/file.txt" {
			t.Errorf("期望 cosPath=subdir/file.txt，实际: %s", cosPath)
		}
		if localPath == "" {
			t.Error("期望 localPath 不为空")
		}
	})

	t.Run("cosPath 以 / 结尾时追加文件路径", func(t *testing.T) {
		fi := fileInfoType{dir: "/tmp", filePath: "file.txt"}
		localPath, cosPath := UploadPathFixed(fi, "prefix/")
		if cosPath != "prefix/file.txt" {
			t.Errorf("期望 cosPath=prefix/file.txt，实际: %s", cosPath)
		}
		if localPath == "" {
			t.Error("期望 localPath 不为空")
		}
	})

	t.Run("cosPath 不以 / 结尾时直接使用 cosPath", func(t *testing.T) {
		fi := fileInfoType{dir: "/tmp", filePath: "file.txt"}
		_, cosPath := UploadPathFixed(fi, "dest/output.txt")
		if cosPath != "dest/output.txt" {
			t.Errorf("期望 cosPath=dest/output.txt，实际: %s", cosPath)
		}
	})
}

func TestGetAbsPath(t *testing.T) {
	t.Run("绝对路径直接返回（加尾部斜杠）", func(t *testing.T) {
		result, err := getAbsPath("/tmp/testdir")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if result == "" {
			t.Error("期望 result 不为空")
		}
	})

	t.Run("相对路径转换为绝对路径", func(t *testing.T) {
		result, err := getAbsPath("relative/path")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if result == "" {
			t.Error("期望 result 不为空")
		}
		if result[0] != '/' {
			t.Errorf("期望绝对路径以 / 开头，实际: %s", result)
		}
	})
}

func TestCreateParentDirectory(t *testing.T) {
	t.Run("创建父目录成功", func(t *testing.T) {
		tmpDir := os.TempDir()
		filePath := tmpDir + "/coscli-test-parent/subdir/file.txt"
		defer os.RemoveAll(tmpDir + "/coscli-test-parent")

		err := createParentDirectory(filePath)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		// 验证父目录已创建
		if _, err := os.Stat(tmpDir + "/coscli-test-parent/subdir"); os.IsNotExist(err) {
			t.Error("期望父目录已创建，但不存在")
		}
	})
}
