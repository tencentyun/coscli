package util

import (
	"testing"
)

func TestEncryptAndDecryptSecret(t *testing.T) {
	t.Run("加密后解密还原原始字符串", func(t *testing.T) {
		original := "my-secret-key-12345"
		encoded, err := EncryptSecret(original)
		if err != nil {
			t.Fatalf("EncryptSecret 失败: %v", err)
		}
		if encoded == "" {
			t.Error("期望 encoded 不为空")
		}
		if encoded == original {
			t.Error("期望 encoded 与原始字符串不同")
		}

		decoded, err := DecryptSecret(encoded)
		if err != nil {
			t.Fatalf("DecryptSecret 失败: %v", err)
		}
		if decoded != original {
			t.Errorf("期望解密后 %q，实际 %q", original, decoded)
		}
	})

	t.Run("空字符串加密解密", func(t *testing.T) {
		original := ""
		encoded, err := EncryptSecret(original)
		if err != nil {
			t.Fatalf("EncryptSecret 失败: %v", err)
		}
		decoded, err := DecryptSecret(encoded)
		if err != nil {
			t.Fatalf("DecryptSecret 失败: %v", err)
		}
		if decoded != original {
			t.Errorf("期望解密后 %q，实际 %q", original, decoded)
		}
	})

	t.Run("长字符串加密解密", func(t *testing.T) {
		original := "test-long-string-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
		encoded, err := EncryptSecret(original)
		if err != nil {
			t.Fatalf("EncryptSecret 失败: %v", err)
		}
		decoded, err := DecryptSecret(encoded)
		if err != nil {
			t.Fatalf("DecryptSecret 失败: %v", err)
		}
		if decoded != original {
			t.Errorf("期望解密后 %q，实际 %q", original, decoded)
		}
	})

	t.Run("DecryptSecret 传入非 base64 字符串时返回错误", func(t *testing.T) {
		_, err := DecryptSecret("not-valid-base64!!!")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("多次加密同一字符串结果相同（ECB 模式确定性）", func(t *testing.T) {
		original := "test-secret"
		encoded1, err := EncryptSecret(original)
		if err != nil {
			t.Fatalf("第一次 EncryptSecret 失败: %v", err)
		}
		encoded2, err := EncryptSecret(original)
		if err != nil {
			t.Fatalf("第二次 EncryptSecret 失败: %v", err)
		}
		if encoded1 != encoded2 {
			t.Errorf("期望两次加密结果相同，实际 %q != %q", encoded1, encoded2)
		}
	})
}

func TestAesTool(t *testing.T) {
	t.Run("ECB 模式加密解密", func(t *testing.T) {
		tool := NewAesTool([]byte(AesKey), AesBlockSize, ECB)
		plaintext := []byte("hello world test")
		encrypted, err := tool.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encrypt 失败: %v", err)
		}
		decrypted, err := tool.Decrypt(encrypted)
		if err != nil {
			t.Fatalf("Decrypt 失败: %v", err)
		}
		if string(decrypted) != string(plaintext) {
			t.Errorf("期望 %q，实际 %q", string(plaintext), string(decrypted))
		}
	})

	t.Run("padding 后长度是 AES 块大小的倍数", func(t *testing.T) {
		tool := NewAesTool([]byte(AesKey), AesBlockSize, ECB)
		// 测试不同长度的输入
		for _, length := range []int{1, 15, 16, 17, 31, 32, 33} {
			input := make([]byte, length)
			padded := tool.padding(input)
			if len(padded)%16 != 0 {
				t.Errorf("padding 后长度 %d 不是 16 的倍数（输入长度 %d）", len(padded), length)
			}
		}
	})
}
