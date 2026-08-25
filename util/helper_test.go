package util

import (
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestGetThreadNumByPartSize(t *testing.T) {
	tests := []struct {
		name        string
		totalSize   int64
		partSize    int64
		rateLimitMB float32
		maxThread   int
		expectedMin int
		expectedMax int
	}{
		// ========== 档位测试（rateLimitMB=0, maxThread=0 默认 32） ==========
		// partNum < 2 → threadNum = 1
		{"小文件（1 个分片）", 10 * 1024 * 1024, 100, 0, 0, 1, 1},
		// partNum < 4 → threadNum = 2
		{"2-3 个分片", 200 * 1024 * 1024, 100, 0, 0, 2, 2},
		// partNum <= 20 → threadNum = 4
		{"5-20 个分片", 500 * 1024 * 1024, 100, 0, 0, 4, 4},
		// partNum <= 300 → threadNum = 8
		{"21-300 个分片", 3000 * 1024 * 1024, 100, 0, 0, 8, 8},
		// partNum <= 500 → threadNum = 12
		{"301-500 个分片", 40000 * 1024 * 1024, 100, 0, 0, 12, 12},
		// partNum <= 2000 → threadNum = 20
		{"501-2000 个分片", 150000 * 1024 * 1024, 100, 0, 0, 20, 20},
		// partNum > 2000 → threadNum = 32（受默认 maxThread=32 封顶）
		{"超过 2000 个分片（封顶 32）", 500000 * 1024 * 1024, 100, 0, 0, 32, 32},

		// ========== 带宽感知测试 ==========
		// rateLimit=16MB/s → 16/8+1=3 线程，小于档位 8（21-300 档）
		{"带宽限制 16MB/s 压制档位", 3000 * 1024 * 1024, 100, 16, 0, 3, 3},
		// rateLimit=100MB/s → 100/8+1=13 线程，小于档位 20（501-2000 档）
		{"带宽限制 100MB/s 压制档位", 150000 * 1024 * 1024, 100, 100, 0, 13, 13},
		// rateLimit=1000MB/s → 125 线程，但封顶 maxThread=32
		{"带宽极大时被 maxThread 封顶", 500000 * 1024 * 1024, 100, 1000, 0, 32, 32},

		// ========== 用户自定义 maxThread 测试 ==========
		// maxThread=16 封顶到 16
		{"maxThread=16 压低默认档位 32", 500000 * 1024 * 1024, 100, 0, 16, 16, 16},
		// maxThread=64 允许超过默认 32 的档位
		{"maxThread=64 档位不超时保持档位值", 500000 * 1024 * 1024, 100, 0, 64, 32, 32},
		// maxThread=1 极端压缩
		{"maxThread=1 极端压缩", 3000 * 1024 * 1024, 100, 0, 1, 1, 1},
		// maxThread<=0 使用默认 32
		{"maxThread=0 使用默认 32", 500000 * 1024 * 1024, 100, 0, 0, 32, 32},
		{"maxThread=-1 使用默认 32", 500000 * 1024 * 1024, 100, 0, -1, 32, 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getThreadNumByPartSize(tt.totalSize, tt.partSize, tt.rateLimitMB, tt.maxThread)
			if err != nil {
				t.Fatalf("期望无错误，但得到: %v", err)
			}
			if result < tt.expectedMin || result > tt.expectedMax {
				t.Errorf("getThreadNumByPartSize(size=%d, part=%d, rate=%.0f, max=%d): 期望 [%d, %d]，实际 %d",
					tt.totalSize, tt.partSize, tt.rateLimitMB, tt.maxThread,
					tt.expectedMin, tt.expectedMax, result)
			}
		})
	}
}

func TestIsSDKHandledError(t *testing.T) {
	t.Run("nil 错误返回 false", func(t *testing.T) {
		if isSDKHandledError(nil) {
			t.Error("期望 nil 返回 false")
		}
	})

	t.Run("5xx 错误（500）返回 true", func(t *testing.T) {
		cosErr := &cos.ErrorResponse{
			Response: &http.Response{StatusCode: 500},
			Code:     "InternalError",
			Message:  "server error",
		}
		if !isSDKHandledError(cosErr) {
			t.Error("期望 5xx(500) 返回 true")
		}
	})

	t.Run("5xx 错误（503）返回 true", func(t *testing.T) {
		cosErr := &cos.ErrorResponse{
			Response: &http.Response{StatusCode: 503},
		}
		if !isSDKHandledError(cosErr) {
			t.Error("期望 5xx(503) 返回 true")
		}
	})

	t.Run("4xx 错误（403）返回 false", func(t *testing.T) {
		cosErr := &cos.ErrorResponse{
			Response: &http.Response{StatusCode: 403},
			Code:     "AccessDenied",
		}
		if isSDKHandledError(cosErr) {
			t.Error("期望 4xx(403) 返回 false")
		}
	})

	t.Run("4xx 错误（429 限频）返回 false", func(t *testing.T) {
		cosErr := &cos.ErrorResponse{
			Response: &http.Response{StatusCode: 429},
		}
		if isSDKHandledError(cosErr) {
			t.Error("期望 4xx(429) 返回 false")
		}
	})

	t.Run("ErrorResponse.Response 为 nil 时返回 false", func(t *testing.T) {
		cosErr := &cos.ErrorResponse{
			Response: nil,
			Code:     "UnknownError",
		}
		if isSDKHandledError(cosErr) {
			t.Error("期望 Response=nil 时返回 false（无法判断状态码）")
		}
	})

	t.Run("net.Error（超时）返回 false（SDK 在 body 为 io.Reader 时不重试，应用层需兜底）", func(t *testing.T) {
		netErr := &net.OpError{
			Op:  "dial",
			Err: &timeoutError{},
		}
		if isSDKHandledError(netErr) {
			t.Error("期望 net.Error 返回 false，让应用层兜底重试")
		}
	})

	t.Run("普通错误返回 false", func(t *testing.T) {
		err := fmt.Errorf("some random application error")
		if isSDKHandledError(err) {
			t.Error("期望普通错误返回 false")
		}
	})

	t.Run("wrapped 5xx 错误通过 errors.As 识别", func(t *testing.T) {
		cosErr := &cos.ErrorResponse{
			Response: &http.Response{StatusCode: 502},
		}
		wrapped := fmt.Errorf("upload failed: %w", cosErr)
		if !isSDKHandledError(wrapped) {
			t.Error("期望 wrapped 5xx 错误返回 true")
		}
	})
}

// timeoutError 辅助类型，实现 net.Error 接口
type timeoutError struct{}

func (e *timeoutError) Error() string   { return "timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }
