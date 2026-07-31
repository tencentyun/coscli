package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetContent(t *testing.T) {
	t.Run("非 file:// 前缀时直接返回输入内容", func(t *testing.T) {
		input := `{"key":"value"}`
		result, err := GetContent(input)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if string(result) != input {
			t.Errorf("期望 %q，实际 %q", input, string(result))
		}
	})

	t.Run("file:// 前缀时读取本地文件", func(t *testing.T) {
		// 创建临时文件
		tmpFile, err := os.CreateTemp("", "coscli-test-*.json")
		if err != nil {
			t.Fatalf("创建临时文件失败: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		content := `{"version":"1.0"}`
		if _, err := tmpFile.WriteString(content); err != nil {
			t.Fatalf("写入临时文件失败: %v", err)
		}
		tmpFile.Close()

		result, err := GetContent("file://" + tmpFile.Name())
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if string(result) != content {
			t.Errorf("期望 %q，实际 %q", content, string(result))
		}
	})

	t.Run("file:// 前缀但文件不存在时返回错误", func(t *testing.T) {
		_, err := GetContent("file:///nonexistent/path/file.json")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("file:// 前缀但路径是目录时返回错误", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "coscli-test-dir-*")
		if err != nil {
			t.Fatalf("创建临时目录失败: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		_, err = GetContent("file://" + tmpDir)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("file:// 前缀但文件为空时返回错误", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "coscli-test-empty-*.json")
		if err != nil {
			t.Fatalf("创建临时文件失败: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.Close() // 不写入任何内容

		_, err = GetContent("file://" + tmpFile.Name())
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})
}

func TestParseContent(t *testing.T) {
	type testStruct struct {
		Key   string `json:"key" xml:"Key"`
		Value string `json:"value" xml:"Value"`
	}

	t.Run("空内容时返回错误", func(t *testing.T) {
		var target testStruct
		err := ParseContent([]byte{}, &target, ContentTypeInventory)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("Policy 类型：JSON 格式解析成功", func(t *testing.T) {
		var target testStruct
		err := ParseContent([]byte(`{"key":"hello","value":"world"}`), &target, ContentTypePolicy)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if target.Key != "hello" {
			t.Errorf("期望 Key=hello，实际: %s", target.Key)
		}
	})

	t.Run("Policy 类型：非 JSON 格式返回错误", func(t *testing.T) {
		var target testStruct
		err := ParseContent([]byte(`<Key>hello</Key>`), &target, ContentTypePolicy)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("非 Policy 类型：JSON 格式解析成功", func(t *testing.T) {
		var target testStruct
		err := ParseContent([]byte(`{"key":"foo","value":"bar"}`), &target, ContentTypeInventory)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if target.Key != "foo" {
			t.Errorf("期望 Key=foo，实际: %s", target.Key)
		}
	})

	t.Run("非 Policy 类型：XML 格式解析成功", func(t *testing.T) {
		var target testStruct
		err := ParseContent([]byte(`<testStruct><Key>xml-key</Key><Value>xml-val</Value></testStruct>`), &target, ContentTypeInventory)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if target.Key != "xml-key" {
			t.Errorf("期望 Key=xml-key，实际: %s", target.Key)
		}
	})

	t.Run("非 Policy 类型：既非 JSON 也非 XML 时返回错误", func(t *testing.T) {
		var target testStruct
		err := ParseContent([]byte(`plain text content`), &target, ContentTypeInventory)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})
}

func TestGetContentWithRelativePath(t *testing.T) {
	t.Run("file:// 前缀相对路径也能正确读取", func(t *testing.T) {
		// 在当前目录创建临时文件
		tmpFile := filepath.Join(os.TempDir(), "coscli-relative-test.json")
		if err := os.WriteFile(tmpFile, []byte(`{"test":true}`), 0644); err != nil {
			t.Fatalf("创建临时文件失败: %v", err)
		}
		defer os.Remove(tmpFile)

		result, err := GetContent("file://" + tmpFile)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if string(result) != `{"test":true}` {
			t.Errorf("期望 {\"test\":true}，实际 %s", string(result))
		}
	})
}
