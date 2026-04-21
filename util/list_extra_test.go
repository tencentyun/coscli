package util

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/olekukonko/tablewriter"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestTableRender(t *testing.T) {
	t.Run("RenderNum 未达到阈值时不渲染", func(t *testing.T) {
		lsCounter := &LsCounter{
			RenderNum: OfsMaxRenderNum - 1,
		}
		lsCounter.Table = tablewriter.NewWriter(os.Stdout)
		tableRender(lsCounter)
		// RenderNum 未变化，说明没有触发渲染
		if lsCounter.RenderNum != OfsMaxRenderNum-1 {
			t.Errorf("期望 RenderNum=%d，实际 %d", OfsMaxRenderNum-1, lsCounter.RenderNum)
		}
	})

	t.Run("RenderNum 达到阈值时触发渲染并重置", func(t *testing.T) {
		lsCounter := &LsCounter{
			RenderNum: OfsMaxRenderNum,
		}
		lsCounter.Table = tablewriter.NewWriter(os.Stdout)
		tableRender(lsCounter)
		// 渲染后 RenderNum 重置为 0
		if lsCounter.RenderNum != 0 {
			t.Errorf("期望 RenderNum=0，实际 %d", lsCounter.RenderNum)
		}
	})
}

func TestGetFilesAndDirs(t *testing.T) {
	// GetFilesAndDirs 内部调用 GetObjectsListIterator → Bucket.Get（已全局打桩）
	// GetObjectsListIterator 直接调用 Bucket.Get，不经过 tryGetObjects 重试

	t.Run("Bucket.Get 失败时返回错误", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return nil, nil, fmt.Errorf("mock get error")
		}
		_, err := GetFilesAndDirs(newTestClient(), "prefix/", "", "", "")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
		mockBucketGetFunc = nil
	})

	t.Run("成功返回文件列表（无子目录）", func(t *testing.T) {
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents: []cos.Object{
					{Key: "prefix/file1.txt"},
					{Key: "prefix/file2.txt"},
				},
				CommonPrefixes: []string{},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		files, err := GetFilesAndDirs(newTestClient(), "prefix/", "", "", "")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		// 包含 cosDir 本身 + 2 个文件
		if len(files) < 2 {
			t.Errorf("期望至少 2 个文件，实际 %d", len(files))
		}
		mockBucketGetFunc = nil
	})
}
