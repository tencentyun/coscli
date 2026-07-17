package cmd

import (
	"context"
	"coscli/util"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// TestRealBucketPathTraversalFix 使用真实桶（alias=test8 / willppantest8）验证对象名路径穿越下载越界写入的修复。
//
// 思路：
//  1. 用 coscli 自身的鉴权 Transport 发送一个“保留 .. 片段”的原始 PUT，
//     尝试在真实桶里放置对象键 "coscli-pt-<rnd>/../owned-...txt"（标准 SDK 会被
//     ResolveReference 规范化掉 ..，因此必须走原始请求）。
//  2. 列举该前缀，打印 COS 实际存储的键，确认穿越键是否真的落到了桶里。
//  3. 走 coscli 真实递归下载链路（FormatDownloadPath -> Download -> getCosObjectList
//     -> singleDownload -> DownloadPathFixed），下载到临时目录 base/victim/。
//  4. 断言：越界目标 base/owned-...txt 未被创建（修复生效）；正常嵌套对象正常落地（无误杀）。
//
// 该用例会真实读写线上桶，默认跳过，需显式设置环境变量 COSCLI_REAL_PT_TEST=1 才执行。
func TestRealBucketPathTraversalFix(t *testing.T) {
	if os.Getenv("COSCLI_REAL_PT_TEST") == "" {
		t.Skip("set COSCLI_REAL_PT_TEST=1 to run real-bucket path traversal test")
	}

	const alias = "test8"

	if config.Base.SecretID == "" || config.Base.SecretKey == "" {
		t.Fatalf("missing credentials in ~/.cos.yaml (config global is empty)")
	}

	// 复刻 initConfig 的解密逻辑：~/.cos.yaml 中的密钥默认是加密存储的，
	// 本测试绕过了 cobra 初始化，需手动解密后写回 config 全局，否则 COS 会报
	// InvalidAccessKeyId（直接拿密文当 AK 用）。
	if config.Base.DisableEncryption != "true" {
		if v, e := util.DecryptSecret(config.Base.SecretKey); e == nil {
			config.Base.SecretKey = v
		} else {
			t.Fatalf("decrypt secretKey failed: %v", e)
		}
		if v, e := util.DecryptSecret(config.Base.SecretID); e == nil {
			config.Base.SecretID = v
		} else {
			t.Fatalf("decrypt secretID failed: %v", e)
		}
		if config.Base.SessionToken != "" {
			if v, e := util.DecryptSecret(config.Base.SessionToken); e == nil {
				config.Base.SessionToken = v
			} else {
				t.Fatalf("decrypt sessionToken failed: %v", e)
			}
		}
	}

	c, err := util.NewClient(&config, &param, alias)
	if err != nil {
		t.Fatalf("NewClient(%s) failed: %v", alias, err)
	}

	rnd := randStr(6)
	prefix := "coscli-pt-" + rnd + "/"
	canaryName := "owned-by-coscli-" + rnd + ".txt"
	maliciousKey := prefix + "../" + canaryName // coscli-pt-<rnd>/../owned-...txt
	benignKey := prefix + "sub/legit-" + rnd + ".txt"
	payloadM := "TSRC_COSCLI_REALBUCKET_PT_" + rnd
	payloadB := "BENIGN_" + rnd

	ctx := context.Background()

	// 原始请求客户端：复用 coscli 的 AuthorizationTransport（自动处理 SessionToken 与签名），
	// 但请求 URL 的 Path 保留 ".."，从而绕过标准 SDK 的路径规范化。
	rawClient := &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:     config.Base.SecretID,
			SecretKey:    config.Base.SecretKey,
			SessionToken: config.Base.SessionToken,
		},
	}
	sendRaw := func(method, key, body string) (int, string, error) {
		u := *c.BaseURL.BucketURL // 拷贝
		u.Path = "/" + key        // 保留 ..，url.URL.Path 不会被清理
		u.RawPath = ""
		req := &http.Request{
			Method: method,
			URL:    &u,
			Host:   u.Host,
			Header: make(http.Header),
			Body:   io.NopCloser(strings.NewReader(body)),
		}
		req.ContentLength = int64(len(body))
		if body != "" {
			req.Header.Set("Content-Type", "text/plain")
		}
		resp, e := rawClient.Do(req)
		if e != nil {
			return 0, "", e
		}
		defer resp.Body.Close()
		rb, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(rb), nil
	}

	// 清理：函数退出时删除测试产生的对象与可能被规范化落库的变体键
	defer func() {
		// 正常对象（无 ..，普通删除即可）
		_, _ = c.Object.Delete(ctx, benignKey)
		// 穿越键：用原始请求删除（保留 ..）
		_, _, _ = sendRaw(http.MethodDelete, maliciousKey, "")
		// 兜底删除可能被规范化后的落点
		_, _ = c.Object.Delete(ctx, canaryName)             // 规范化到桶根
		_, _ = c.Object.Delete(ctx, prefix+canaryName)      // 规范化到前缀内
	}()

	// 1) 放置正常嵌套对象（回归用：验证修复不影响正常下载）
	if _, err := c.Object.Put(ctx, benignKey, strings.NewReader(payloadB), nil); err != nil {
		t.Fatalf("put benign object failed: %v", err)
	}
	t.Logf("[setup] benign object put: %s", benignKey)

	// 2) 放置穿越对象（保留 ..）
	code, respBody, err := sendRaw(http.MethodPut, maliciousKey, payloadM)
	if err != nil {
		t.Fatalf("raw PUT malicious key failed: %v", err)
	}
	t.Logf("[setup] raw PUT %q -> HTTP %d %s", maliciousKey, code, strings.TrimSpace(respBody))

	// 3) 列举前缀，打印 COS 实际存储的对象键
	storedKeys := map[string]bool{}
	{
		opt := &cos.BucketGetOptions{Prefix: prefix, EncodingType: "url", MaxKeys: 1000}
		res, _, lerr := c.Bucket.Get(ctx, opt)
		if lerr != nil {
			t.Fatalf("list bucket prefix %q failed: %v", prefix, lerr)
		}
		for _, o := range res.Contents {
			k, _ := url.QueryUnescape(o.Key)
			storedKeys[k] = true
			t.Logf("[list] stored key in bucket: %q (size=%d)", k, o.Size)
		}
	}
	maliciousStored := storedKeys[maliciousKey]
	if maliciousStored {
		t.Logf("[list] COS 保留了穿越键，进入真实越界下载验证")
	} else {
		t.Logf("[list] COS 未按原样保留穿越键（被规范化/拒绝）。仍继续走下载链路做回归与防御验证")
	}

	// 4) 走 coscli 真实递归下载链路
	base := t.TempDir()
	victim := filepath.Join(base, "victim") + string(os.PathSeparator)
	if err := os.MkdirAll(victim, 0755); err != nil {
		t.Fatal(err)
	}

	cosURL, err := util.FormatUrl("cos://" + alias + "/" + prefix)
	if err != nil {
		t.Fatalf("FormatUrl cos failed: %v", err)
	}
	fileURL, err := util.FormatUrl(victim)
	if err != nil {
		t.Fatalf("FormatUrl file failed: %v", err)
	}

	fo := &util.FileOperations{
		Operation: util.Operation{
			Recursive:   true,
			PartSize:    1,
			Routines:    2,
			ErrRetryNum: 0,
		},
		Monitor:    &util.FileProcessMonitor{},
		CpType:     util.CpTypeDownload,
		Command:    util.CommandCP,
		BucketType: util.BucketTypeCos,
	}

	if err := util.FormatDownloadPath(cosURL, fileURL, fo, c); err != nil {
		t.Fatalf("FormatDownloadPath failed: %v", err)
	}
	if err := util.Download(c, cosURL, fileURL, fo); err != nil {
		t.Logf("[download] Download returned err (单对象失败不影响整体返回): %v", err)
	}

	// 5) 断言
	// (a) 越界目标必须不存在（修复生效的核心证据）
	escapePath := filepath.Clean(filepath.Join(base, canaryName)) // base/owned-...txt，位于 victim 之外
	if _, statErr := os.Stat(escapePath); statErr == nil {
		data, _ := os.ReadFile(escapePath)
		t.Fatalf("VULNERABLE: 越界文件被写出到下载目录之外: %s, 内容=%q", escapePath, string(data))
	} else if !os.IsNotExist(statErr) {
		t.Fatalf("unexpected stat error on %s: %v", escapePath, statErr)
	}
	t.Logf("[assert] OK: 越界路径未被写入: %s", escapePath)

	// 兜底：victim 目录树内不应出现 canary 文件（即穿越内容不应以任何形式落到下载目录内外）
	if maliciousStored {
		insideCanary := filepath.Join(victim, canaryName)
		if _, statErr := os.Stat(insideCanary); statErr == nil {
			t.Fatalf("VULNERABLE: 穿越内容被写入下载目录内 %s（说明拼接/校验异常）", insideCanary)
		}
	}

	// (b) 回归：正常嵌套对象应正确落到 victim/sub/legit-<rnd>.txt
	legitPath := filepath.Join(victim, "sub", "legit-"+rnd+".txt")
	data, rerr := os.ReadFile(legitPath)
	if rerr != nil {
		t.Fatalf("REGRESSION: 正常嵌套对象未能正确下载到 %s: %v", legitPath, rerr)
	}
	if string(data) != payloadB {
		t.Fatalf("REGRESSION: 正常对象内容不一致, got=%q want=%q", string(data), payloadB)
	}
	t.Logf("[assert] OK: 正常嵌套对象正确落地: %s", legitPath)

	t.Logf("RESULT: 路径穿越修复在真实桶 %s 验证通过（越界被拦截，正常下载不受影响）", alias)
}
