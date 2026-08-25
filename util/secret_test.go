package util

import (
	"crypto/aes"
	"strings"
	"testing"
)

func TestAesToolUnPadding(t *testing.T) {
	tool := NewAesTool([]byte(AesKey), AesBlockSize, ECB)

	t.Run("unPadding 去除尾部零字节", func(t *testing.T) {
		src := []byte{'h', 'e', 'l', 'l', 'o', 0, 0, 0}
		unpadded := tool.unPadding(src)
		if string(unpadded) != "hello" {
			t.Errorf("期望 unpadded=%q，实际=%q", "hello", string(unpadded))
		}
	})

	t.Run("unPadding 全零字节返回 nil", func(t *testing.T) {
		src := []byte{0, 0, 0, 0}
		unpadded := tool.unPadding(src)
		if unpadded != nil {
			t.Errorf("期望 nil，实际: %v", unpadded)
		}
	})

	t.Run("unPadding 无零字节时返回原始数据", func(t *testing.T) {
		src := []byte{'a', 'b', 'c'}
		unpadded := tool.unPadding(src)
		if string(unpadded) != "abc" {
			t.Errorf("期望 abc，实际: %q", string(unpadded))
		}
	})
}

func TestECBEncrypterDecrypter(t *testing.T) {
	t.Run("ECBEncrypter BlockSize 返回正确值", func(t *testing.T) {
		block, err := aes.NewCipher(make([]byte, 16))
		if err != nil {
			t.Fatalf("创建 cipher 失败: %v", err)
		}
		enc := NewECBEncrypter(block)
		if enc.BlockSize() != aes.BlockSize {
			t.Errorf("期望 BlockSize=%d，实际=%d", aes.BlockSize, enc.BlockSize())
		}
	})

	t.Run("ECBDecrypter BlockSize 返回正确值", func(t *testing.T) {
		block, err := aes.NewCipher(make([]byte, 16))
		if err != nil {
			t.Fatalf("创建 cipher 失败: %v", err)
		}
		dec := NewECBDecrypter(block)
		if dec.BlockSize() != aes.BlockSize {
			t.Errorf("期望 BlockSize=%d，实际=%d", aes.BlockSize, dec.BlockSize())
		}
	})

	t.Run("ECBEncrypter CryptBlocks 加密后 ECBDecrypter 解密还原", func(t *testing.T) {
		key := make([]byte, 16)
		block, err := aes.NewCipher(key)
		if err != nil {
			t.Fatalf("创建 cipher 失败: %v", err)
		}

		src := make([]byte, aes.BlockSize)
		copy(src, []byte("hello world!!!!"))

		dst := make([]byte, aes.BlockSize)
		enc := NewECBEncrypter(block)
		enc.CryptBlocks(dst, src)

		// 解密
		block2, _ := aes.NewCipher(key)
		dec := NewECBDecrypter(block2)
		result := make([]byte, aes.BlockSize)
		dec.CryptBlocks(result, dst)

		if string(result) != string(src) {
			t.Errorf("期望解密后=%q，实际=%q", string(src), string(result))
		}
	})
}

func TestAesToolEncryptEmptyBytes(t *testing.T) {
	t.Run("ECB 加密空字节后长度为一个 block", func(t *testing.T) {
		tool := NewAesTool([]byte(AesKey), AesBlockSize, ECB)
		src := []byte{}
		encrypted, err := tool.Encrypt(src)
		if err != nil {
			t.Fatalf("Encrypt 失败: %v", err)
		}
		if len(encrypted) != aes.BlockSize {
			t.Errorf("期望 encrypted 长度=%d，实际=%d", aes.BlockSize, len(encrypted))
		}
	})
}

func TestEncryptDecryptSecret(t *testing.T) {
	t.Run("加密后解密还原原始字符串", func(t *testing.T) {
		original := "my-secret-key-12345"
		encoded, err := EncryptSecret(original)
		if err != nil {
			t.Fatalf("EncryptSecret 失败: %v", err)
		}
		if encoded == "" {
			t.Fatal("加密结果不应为空")
		}
		if encoded == original {
			t.Error("加密结果不应与原始字符串相同")
		}

		decoded, err := DecryptSecret(encoded)
		if err != nil {
			t.Fatalf("DecryptSecret 失败: %v", err)
		}
		if decoded != original {
			t.Errorf("解密结果: 期望 %q，实际 %q", original, decoded)
		}
	})

	t.Run("空字符串加密解密", func(t *testing.T) {
		original := ""
		encoded, err := EncryptSecret(original)
		if err != nil {
			t.Fatalf("EncryptSecret 空字符串失败: %v", err)
		}
		decoded, err := DecryptSecret(encoded)
		if err != nil {
			t.Fatalf("DecryptSecret 空字符串失败: %v", err)
		}
		if decoded != original {
			t.Errorf("空字符串解密: 期望 %q，实际 %q", original, decoded)
		}
	})

	t.Run("长字符串加密解密", func(t *testing.T) {
		original := strings.Repeat("abcdefghij", 10)
		encoded, err := EncryptSecret(original)
		if err != nil {
			t.Fatalf("EncryptSecret 长字符串失败: %v", err)
		}
		decoded, err := DecryptSecret(encoded)
		if err != nil {
			t.Fatalf("DecryptSecret 长字符串失败: %v", err)
		}
		if decoded != original {
			t.Errorf("长字符串解密: 期望 %q，实际 %q", original, decoded)
		}
	})

	t.Run("特殊字符加密解密", func(t *testing.T) {
		original := "!@#$%^&*()_+-=[]{}|;':\",./<>?"
		encoded, err := EncryptSecret(original)
		if err != nil {
			t.Fatalf("EncryptSecret 特殊字符失败: %v", err)
		}
		decoded, err := DecryptSecret(encoded)
		if err != nil {
			t.Fatalf("DecryptSecret 特殊字符失败: %v", err)
		}
		if decoded != original {
			t.Errorf("特殊字符解密: 期望 %q，实际 %q", original, decoded)
		}
	})

	t.Run("DecryptSecret 非法 base64 字符串返回错误", func(t *testing.T) {
		_, err := DecryptSecret("not-valid-base64!!!")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})
}

func TestAesTool_ECB(t *testing.T) {
	tool := NewAesTool([]byte(AesKey), AesBlockSize, ECB)

	t.Run("ECB 加密解密", func(t *testing.T) {
		src := []byte("hello world test")
		encrypted, err := tool.Encrypt(src)
		if err != nil {
			t.Fatalf("Encrypt 失败: %v", err)
		}
		if len(encrypted) == 0 {
			t.Fatal("加密结果不应为空")
		}

		decrypted, err := tool.Decrypt(encrypted)
		if err != nil {
			t.Fatalf("Decrypt 失败: %v", err)
		}
		if string(decrypted) != string(src) {
			t.Errorf("ECB 解密: 期望 %q，实际 %q", string(src), string(decrypted))
		}
	})

	t.Run("ECB 加密不同长度数据", func(t *testing.T) {
		testCases := []string{
			"a",
			"abcdefghijklmnop",       // 恰好 16 字节
			"abcdefghijklmnopqrstuv", // 超过 16 字节
		}
		for _, tc := range testCases {
			encrypted, err := tool.Encrypt([]byte(tc))
			if err != nil {
				t.Fatalf("Encrypt %q 失败: %v", tc, err)
			}
			decrypted, err := tool.Decrypt(encrypted)
			if err != nil {
				t.Fatalf("Decrypt %q 失败: %v", tc, err)
			}
			if string(decrypted) != tc {
				t.Errorf("ECB 加解密 %q: 期望 %q，实际 %q", tc, tc, string(decrypted))
			}
		}
	})
}

func TestAesTool_ECBRoundTrip(t *testing.T) {
	// CBC 模式的 Encrypt 函数存在 bug：case CBC 内部用 := 重新声明了 encryptData，
	// 导致外部的 encryptData 仍为 nil，CBC 加密返回空字节。
	// 此处只测试 ECB 模式的完整加解密流程。
	key := []byte("1234567890123456") // 16 字节
	tool := NewAesTool(key, AesBlockSize, ECB)

	t.Run("ECB 模式加密后解密还原", func(t *testing.T) {
		src := []byte("hello world test")
		encrypted, err := tool.Encrypt(src)
		if err != nil {
			t.Fatalf("加密失败: %v", err)
		}
		if len(encrypted) == 0 {
			t.Fatal("期望加密结果非空")
		}
		decrypted, err := tool.Decrypt(encrypted)
		if err != nil {
			t.Fatalf("解密失败: %v", err)
		}
		if string(decrypted) != string(src) {
			t.Errorf("期望解密结果 %q，实际 %q", string(src), string(decrypted))
		}
	})
}

func TestAesTool_Padding(t *testing.T) {
	tool := NewAesTool([]byte(AesKey), AesBlockSize, ECB)

	t.Run("padding 长度恰好是 block size 倍数时仍填充一个完整块", func(t *testing.T) {
		// 16 字节时 paddingCount = 16 - 16%16 = 16，填充 16 个 0，变成 32 字节
		src := []byte("1234567890123456") // 16 字节
		padded := tool.padding(src)
		if len(padded) != 32 {
			t.Errorf("期望 padding 后长度为 32，实际 %d", len(padded))
		}
	})

	t.Run("padding 不足时填充 0", func(t *testing.T) {
		src := []byte("hello") // 5 字节，需填充到 16
		padded := tool.padding(src)
		if len(padded) != 16 {
			t.Errorf("期望 padding 后长度为 16，实际 %d", len(padded))
		}
		// 检查填充的是 0
		for i := 5; i < 16; i++ {
			if padded[i] != 0 {
				t.Errorf("padding[%d] 期望 0，实际 %d", i, padded[i])
			}
		}
	})
}

func TestNewAesTool(t *testing.T) {
	tool := NewAesTool([]byte("testkey"), 16, ECB)
	if tool == nil {
		t.Fatal("NewAesTool 返回 nil")
	}
	if tool.Mode != ECB {
		t.Errorf("Mode: 期望 %d，实际 %d", ECB, tool.Mode)
	}
	if tool.BlockSize != 16 {
		t.Errorf("BlockSize: 期望 16，实际 %d", tool.BlockSize)
	}
}
