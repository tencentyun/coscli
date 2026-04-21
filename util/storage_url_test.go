package util

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// mockObjectIsExistFunc 全局 mock 变量，控制 Object.IsExist 行为
// （Object.IsExist 已在 TestMain 中全局打桩）
var mockObjectIsExistFunc func(ctx context.Context, name string, id ...string) (bool, error)

func TestFormatUrl(t *testing.T) {
	t.Run("有效的 cos:// URL（含 object）", func(t *testing.T) {
		url, err := FormatUrl("cos://my-bucket/path/to/file.txt")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if !url.IsCosUrl() {
			t.Error("期望是 COS URL")
		}
		if url.IsFileUrl() {
			t.Error("期望不是 File URL")
		}
		cosUrl := url.(*CosUrl)
		if cosUrl.Bucket != "my-bucket" {
			t.Errorf("期望 Bucket=my-bucket，实际: %s", cosUrl.Bucket)
		}
		if cosUrl.Object != "path/to/file.txt" {
			t.Errorf("期望 Object=path/to/file.txt，实际: %s", cosUrl.Object)
		}
	})

	t.Run("有效的 cos:// URL（只有 bucket）", func(t *testing.T) {
		url, err := FormatUrl("cos://my-bucket")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if !url.IsCosUrl() {
			t.Error("期望是 COS URL")
		}
		cosUrl := url.(*CosUrl)
		if cosUrl.Bucket != "my-bucket" {
			t.Errorf("期望 Bucket=my-bucket，实际: %s", cosUrl.Bucket)
		}
		if cosUrl.Object != "" {
			t.Errorf("期望 Object 为空，实际: %s", cosUrl.Object)
		}
	})

	t.Run("大写 COS:// 前缀也能识别", func(t *testing.T) {
		url, err := FormatUrl("COS://my-bucket/key")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if !url.IsCosUrl() {
			t.Error("期望是 COS URL")
		}
	})

	t.Run("本地文件路径返回 FileUrl", func(t *testing.T) {
		url, err := FormatUrl("/tmp/local/file.txt")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if url.IsCosUrl() {
			t.Error("期望不是 COS URL")
		}
		if !url.IsFileUrl() {
			t.Error("期望是 File URL")
		}
		if url.ToString() != "/tmp/local/file.txt" {
			t.Errorf("期望 ToString()=/tmp/local/file.txt，实际: %s", url.ToString())
		}
	})

	t.Run("相对路径返回 FileUrl", func(t *testing.T) {
		url, err := FormatUrl("relative/path/file.txt")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if url.IsCosUrl() {
			t.Error("期望不是 COS URL")
		}
		if !url.IsFileUrl() {
			t.Error("期望是 File URL")
		}
	})
}

func TestCosUrlToString(t *testing.T) {
	t.Run("有 object 时 ToString 包含 bucket 和 object", func(t *testing.T) {
		cu := &CosUrl{Bucket: "my-bucket", Object: "path/to/file.txt"}
		expected := "cos://my-bucket/path/to/file.txt"
		if cu.ToString() != expected {
			t.Errorf("期望 %s，实际: %s", expected, cu.ToString())
		}
	})

	t.Run("无 object 时 ToString 只包含 bucket", func(t *testing.T) {
		cu := &CosUrl{Bucket: "my-bucket", Object: ""}
		expected := "cos://my-bucket"
		if cu.ToString() != expected {
			t.Errorf("期望 %s，实际: %s", expected, cu.ToString())
		}
	})
}

func TestCosUrlUpdateUrlStr(t *testing.T) {
	t.Run("UpdateUrlStr 更新 bucket 和 object", func(t *testing.T) {
		cu := &CosUrl{}
		cu.UpdateUrlStr("cos://new-bucket/new-object")
		if cu.Bucket != "new-bucket" {
			t.Errorf("期望 Bucket=new-bucket，实际: %s", cu.Bucket)
		}
		if cu.Object != "new-object" {
			t.Errorf("期望 Object=new-object，实际: %s", cu.Object)
		}
	})
}

func TestFileUrlMethods(t *testing.T) {
	t.Run("FileUrl.IsCosUrl 返回 false", func(t *testing.T) {
		fu := FileUrl{}
		if fu.IsCosUrl() {
			t.Error("期望 IsCosUrl()=false")
		}
	})

	t.Run("FileUrl.IsFileUrl 返回 true", func(t *testing.T) {
		fu := FileUrl{}
		if !fu.IsFileUrl() {
			t.Error("期望 IsFileUrl()=true")
		}
	})

	t.Run("FileUrl.UpdateUrlStr 更新 urlStr", func(t *testing.T) {
		fu := &FileUrl{}
		fu.UpdateUrlStr("/new/path/file.txt")
		if fu.ToString() != "/new/path/file.txt" {
			t.Errorf("期望 ToString()=/new/path/file.txt，实际: %s", fu.ToString())
		}
	})
}

func TestCurrentHomeDir(t *testing.T) {
	t.Run("返回非空 home 目录", func(t *testing.T) {
		result := currentHomeDir()
		if result == "" {
			t.Error("期望 homeDir 不为空")
		}
	})
}

func TestFileUrlInit(t *testing.T) {
	t.Run("普通路径直接赋值", func(t *testing.T) {
		fu := &FileUrl{}
		err := fu.Init("/tmp/test.txt")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if fu.ToString() != "/tmp/test.txt" {
			t.Errorf("期望 /tmp/test.txt，实际: %s", fu.ToString())
		}
	})

	t.Run("~/开头的路径展开为 home 目录", func(t *testing.T) {
		fu := &FileUrl{}
		err := fu.Init("~/test.txt")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		result := fu.ToString()
		if result == "~/test.txt" {
			t.Error("期望 ~ 被展开，但仍为 ~/test.txt")
		}
		if len(result) < 2 || result[0] != '/' {
			t.Errorf("期望展开后以 / 开头，实际: %s", result)
		}
	})
}

func TestGetCosUrl(t *testing.T) {
	t.Run("有 bucket 和 object 时返回完整 cos URL", func(t *testing.T) {
		result := getCosUrl("my-bucket", "path/to/file.txt")
		expected := "cos://my-bucket/path/to/file.txt"
		if result != expected {
			t.Errorf("期望 %s，实际: %s", expected, result)
		}
	})

	t.Run("只有 bucket 时返回 cos://bucket", func(t *testing.T) {
		result := getCosUrl("my-bucket", "")
		expected := "cos://my-bucket"
		if result != expected {
			t.Errorf("期望 %s，实际: %s", expected, result)
		}
	})
}

func TestFormatUploadPath(t *testing.T) {
	t.Run("localPath 为空时返回错误", func(t *testing.T) {
		fileUrl := &FileUrl{}
		cosUrl := &CosUrl{Bucket: "my-bucket", Object: "prefix/"}
		fo := &FileOperations{Operation: Operation{Recursive: false}}
		err := FormatUploadPath(fileUrl, cosUrl, fo)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("localPath 不存在时返回错误", func(t *testing.T) {
		fileUrl := &FileUrl{urlStr: "/nonexistent/path/file.txt"}
		cosUrl := &CosUrl{Bucket: "my-bucket", Object: "prefix/"}
		fo := &FileOperations{Operation: Operation{Recursive: false}}
		err := FormatUploadPath(fileUrl, cosUrl, fo)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("localPath 是目录但未指定 --recursive 时返回错误", func(t *testing.T) {
		fileUrl := &FileUrl{urlStr: "/tmp"}
		cosUrl := &CosUrl{Bucket: "my-bucket", Object: "prefix/"}
		fo := &FileOperations{Operation: Operation{Recursive: false}}
		err := FormatUploadPath(fileUrl, cosUrl, fo)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("localPath 是文件且 cosPath 以 / 结尾时拼接文件名", func(t *testing.T) {
		tmpFile := "/tmp/coscli-test-upload-file.txt"
		f, err := os.Create(tmpFile)
		if err != nil {
			t.Fatalf("创建临时文件失败: %v", err)
		}
		f.Close()
		defer os.Remove(tmpFile)

		fileUrl := &FileUrl{urlStr: tmpFile}
		cosUrl := &CosUrl{Bucket: "my-bucket", Object: "prefix/"}
		fo := &FileOperations{Operation: Operation{Recursive: false}}
		err2 := FormatUploadPath(fileUrl, cosUrl, fo)
		if err2 != nil {
			t.Fatalf("期望无错误，但得到: %v", err2)
		}
		// cosPath 应该变成 prefix/coscli-test-upload-file.txt
		if cosUrl.Object != "prefix/coscli-test-upload-file.txt" {
			t.Errorf("期望 cosPath=prefix/coscli-test-upload-file.txt，实际: %s", cosUrl.Object)
		}
	})
}

// createTempFile 创建临时文件，返回清理函数
func createTempFile(path string) (func(), error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	f.Close()
	return func() {
		os.Remove(path)
	}, nil
}

func TestFormatDownloadPath(t *testing.T) {
	fo := &FileOperations{
		Operation:  Operation{Recursive: false},
		BucketType: BucketTypeCos,
	}

	t.Run("localPath 为空时返回错误", func(t *testing.T) {
		fileUrl := &FileUrl{}
		cosUrl := &CosUrl{Bucket: "my-bucket", Object: "file.txt"}
		err := FormatDownloadPath(cosUrl, fileUrl, fo, newTestClient())
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("cosPath 为空且非 recursive 时返回错误", func(t *testing.T) {
		fileUrl := &FileUrl{urlStr: "/tmp/output.txt"}
		cosUrl := &CosUrl{Bucket: "my-bucket", Object: ""}
		err := FormatDownloadPath(cosUrl, fileUrl, fo, newTestClient())
		if err == nil {
			t.Error("期望返回错误（cosPath 是目录），但得到 nil")
		}
	})

	t.Run("cosPath 以 / 结尾且非 recursive 时返回错误", func(t *testing.T) {
		fileUrl := &FileUrl{urlStr: "/tmp/output/"}
		cosUrl := &CosUrl{Bucket: "my-bucket", Object: "prefix/"}
		err := FormatDownloadPath(cosUrl, fileUrl, fo, newTestClient())
		if err == nil {
			t.Error("期望返回错误（cosPath 是目录），但得到 nil")
		}
	})

	t.Run("非 recursive 时对象存在则成功", func(t *testing.T) {
		// Object.IsExist 内部调用 Object.Head，Object.Head 已在 TestMain 中全局打桩
		// 通过 mockHeadFunc 返回 200 模拟对象存在
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
		}
		defer func() { mockHeadFunc = nil }()
		fileUrl := &FileUrl{urlStr: "/tmp/output.txt"}
		cosUrl := &CosUrl{Bucket: "my-bucket", Object: "file.txt"}
		err := FormatDownloadPath(cosUrl, fileUrl, fo, newTestClient())
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("非 recursive 时对象不存在则返回错误", func(t *testing.T) {
		// Object.IsExist 内部调用 Object.Head，返回 404 时 IsExist 返回 (false, nil)
		// FormatDownloadPath 检测到 fileExist=false 时应返回 "cos object not found" 错误
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return nil, &cos.ErrorResponse{Response: &http.Response{StatusCode: 404}}
		}
		defer func() { mockHeadFunc = nil }()
		fileUrl := &FileUrl{urlStr: "/tmp/output.txt"}
		cosUrl := &CosUrl{Bucket: "my-bucket", Object: "nonexistent.txt"}
		err := FormatDownloadPath(cosUrl, fileUrl, fo, newTestClient())
		if err == nil {
			t.Error("期望返回错误（对象不存在），但得到 nil")
		}
	})

	t.Run("Recursive=true 且 cosPath 是目录时成功", func(t *testing.T) {
		foRecursive := &FileOperations{
			Operation:  Operation{Recursive: true},
			BucketType: BucketTypeCos,
		}
		mockBucketGetFunc = func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			return &cos.BucketGetResult{
				Contents: []cos.Object{{Key: "prefix/file.txt"}},
			}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
		}
		tmpDir := os.TempDir()
		cosUrl := &CosUrl{Bucket: "my-bucket", Object: "prefix/"}
		fileUrl := &FileUrl{urlStr: tmpDir + string(os.PathSeparator)}
		err := FormatDownloadPath(cosUrl, fileUrl, foRecursive, newTestClient())
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		mockBucketGetFunc = nil
	})
}

func TestFormatCopyPath(t *testing.T) {
	fo := &FileOperations{
		Operation:  Operation{Recursive: false},
		BucketType: BucketTypeCos,
	}

	t.Run("srcPath 为空且非 recursive 时返回错误", func(t *testing.T) {
		srcUrl := &CosUrl{Bucket: "src-bucket", Object: ""}
		destUrl := &CosUrl{Bucket: "dest-bucket", Object: "dest-file.txt"}
		err := FormatCopyPath(srcUrl, destUrl, fo, newTestClient())
		if err == nil {
			t.Error("期望返回错误（srcPath 是目录），但得到 nil")
		}
	})

	t.Run("srcPath 以 / 结尾且非 recursive 时返回错误", func(t *testing.T) {
		srcUrl := &CosUrl{Bucket: "src-bucket", Object: "prefix/"}
		destUrl := &CosUrl{Bucket: "dest-bucket", Object: "dest-prefix/"}
		err := FormatCopyPath(srcUrl, destUrl, fo, newTestClient())
		if err == nil {
			t.Error("期望返回错误（srcPath 是目录），但得到 nil")
		}
	})

	t.Run("非 recursive 时源对象存在则成功", func(t *testing.T) {
		// Object.IsExist 内部调用 Object.Head，Object.Head 已在 TestMain 中全局打桩
		// 通过 mockHeadFunc 返回 200 模拟对象存在
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
		}
		defer func() { mockHeadFunc = nil }()
		srcUrl := &CosUrl{Bucket: "src-bucket", Object: "src-file.txt"}
		destUrl := &CosUrl{Bucket: "dest-bucket", Object: "dest-file.txt"}
		err := FormatCopyPath(srcUrl, destUrl, fo, newTestClient())
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("非 recursive 时源对象不存在则返回错误", func(t *testing.T) {
		// Object.IsExist 内部调用 Object.Head，返回 404 时 IsExist 返回 (false, nil)
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return nil, &cos.ErrorResponse{Response: &http.Response{StatusCode: 404}}
		}
		defer func() { mockHeadFunc = nil }()
		srcUrl := &CosUrl{Bucket: "src-bucket", Object: "nonexistent.txt"}
		destUrl := &CosUrl{Bucket: "dest-bucket", Object: "dest-file.txt"}
		err := FormatCopyPath(srcUrl, destUrl, fo, newTestClient())
		if err == nil {
			t.Error("期望返回错误（源对象不存在），但得到 nil")
		}
	})

	t.Run("src 不以 / 结尾且 dest 以 / 结尾时拼接文件名", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
		}
		defer func() { mockHeadFunc = nil }()
		srcUrl := &CosUrl{Bucket: "src-bucket", Object: "path/src-file.txt"}
		destUrl := &CosUrl{Bucket: "dest-bucket", Object: "dest-prefix/"}
		err := FormatCopyPath(srcUrl, destUrl, fo, newTestClient())
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		// destPath 应该变成 dest-prefix/src-file.txt
		if destUrl.Object != "dest-prefix/src-file.txt" {
			t.Errorf("期望 destPath=dest-prefix/src-file.txt，实际: %s", destUrl.Object)
		}
	})
}
