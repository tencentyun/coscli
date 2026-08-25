package util

import (
	"errors"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// defaultMaxThreadNum 自动推导分片并发数时的默认上限。
// 原始实现硬编码为 12，对 16GB 以上大文件并发度明显不足；提升到 32 以覆盖大文件场景。
const defaultMaxThreadNum = 32

// singleConnThroughputMB 单连接有效吞吐经验值（MB/s），用于按 RateLimiting 推导线程数。
const singleConnThroughputMB = 8

// getThreadNumByPartSize 根据文件大小、分片大小、带宽限制和上限配置，推导单文件分片并发数。
//
// 参数:
//   - totalSize:      文件总大小（字节）
//   - partSize:       分片大小（MB）
//   - rateLimitMB:    带宽限制（MB/s），0 表示不限速
//   - maxThread:      用户配置的上限（--max-thread-num），<=0 时使用默认 32
//
// 推导顺序:
//   ① 按分片数走档位表得到"档位线程数"
//   ② 若有带宽限制，按每连接 8MB/s 估算上限（过大带宽场景无用，仅对小带宽收敛）
//   ③ 用 maxThread 封顶（默认 32）
//   ④ 保证 >=1 且不超过分片数（避免开出多于分片数的无用线程）
func getThreadNumByPartSize(totalSize, partSize int64, rateLimitMB float32, maxThread int) (int, error) {
	_, partNum, err := cos.SplitSizeIntoChunksToDownload(totalSize, partSize*1024*1024)
	if err != nil {
		return 0, err
	}

	var threadNum int
	switch {
	case partNum < 2:
		threadNum = 1
	case partNum < 4:
		threadNum = 2
	case partNum <= 20:
		threadNum = 4
	case partNum <= 300:
		threadNum = 8
	case partNum <= 500:
		threadNum = 12
	case partNum <= 2000:
		threadNum = 20
	default:
		threadNum = 32
	}

	// 带宽感知：按每连接 8MB/s 估算，避免带宽受限时开过多无效线程
	if rateLimitMB > 0 {
		byBandwidth := int(rateLimitMB)/singleConnThroughputMB + 1
		if byBandwidth < threadNum {
			threadNum = byBandwidth
		}
	}

	// 用户配置的上限封顶（默认 32）
	if maxThread <= 0 {
		maxThread = defaultMaxThreadNum
	}
	if threadNum > maxThread {
		threadNum = maxThread
	}

	// 兜底：至少 1，且不超过分片数
	if threadNum < 1 {
		threadNum = 1
	}
	if threadNum > partNum && partNum > 0 {
		threadNum = partNum
	}

	return threadNum, nil
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
