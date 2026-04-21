package util

import (
	"errors"

	"github.com/tencentyun/cos-go-sdk-v5"
)

func getThreadNumByPartSize(totalSize, partSize int64) (int, error) {
	var threadNum int
	_, partNum, err := cos.SplitSizeIntoChunksToDownload(totalSize, partSize*1024*1024)
	if err != nil {
		return threadNum, err
	}

	if partNum < 2 {
		threadNum = 1
	} else if partNum < 4 {
		threadNum = 2
	} else if partNum <= 20 {
		threadNum = 4
	} else if partNum <= 300 {
		threadNum = 8
	} else if partNum <= 500 {
		threadNum = 10
	} else {
		threadNum = 12
	}
	return threadNum, err
}

// isSDKHandledError 判断错误是否为 SDK 层已经充分重试过的错误类型（仅限 5xx 服务端错误）。
//
// 背景：
//   cos-go-sdk-v5 的 CheckRetrieable 会对 5xx 响应自动重试（默认 10 次）。
//   coscli 应用层再做一次重试（默认 5 次）会产生叠加放大效应，最坏情况 66 次请求。
//   因此应用层针对 5xx 错误不再叠加重试。
//
// 关于网络错误为何不在此判断：
//   SDK 的 doRetry 在请求 body 为 io.Reader（典型如上传分片、PutObject）时，
//   直接 return，不进入重试循环（因为 Reader 无法重放）。此时若应用层也放弃
//   重试，整个请求将彻底没有重试机会。因此网络错误交给应用层重试兜底更安全。
//
// 返回 true 表示 SDK 已做过 HTTP 级重试，应用层应放弃继续重试。
func isSDKHandledError(err error) bool {
	if err == nil {
		return false
	}

	// 服务端返回的结构化错误：按 StatusCode 粗粒度判断 5xx
	var cosErr *cos.ErrorResponse
	if errors.As(err, &cosErr) && cosErr.Response != nil {
		sc := cosErr.Response.StatusCode
		if sc >= 500 && sc < 600 {
			return true
		}
	}

	return false
}
