package cmd

import (
	"context"
	"coscli/util"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestCpCmd(t *testing.T) {
	fmt.Println("TestCpCmd")
	testBucket1 = randStr(8)
	testAlias1 = testBucket1 + "-alias"
	testBucket2 = randStr(8)
	testAlias2 = testBucket2 + "-alias"
	testVersionBucket = randStr(8)
	testVersionBucketAlias = testVersionBucket + "-alias"
	setUp(testBucket1, testAlias1, testEndpoint, false, false)
	defer tearDown(testBucket1, testAlias1, testEndpoint, false)
	setUp(testBucket2, testAlias2, testEndpoint, false, false)
	defer tearDown(testBucket2, testAlias2, testEndpoint, false)
	setUp(testVersionBucket, testVersionBucketAlias, testEndpoint, false, true)
	defer tearDown(testVersionBucket, testVersionBucketAlias, testEndpoint, true)
	c, _ := util.NewClient(&config, &param, testVersionBucketAlias)
	clearCmd()
	cmd := rootCmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	genDir(testDir, 3)
	defer delDir(testDir)
	Convey("Test coscli cp", t, func() {
		Convey("上传单个小文件到多版本桶", func() {
			clearCmd()
			cmd := rootCmd
			localFileName := fmt.Sprintf("%s/small-file/0", testDir)
			cosFileName := fmt.Sprintf("cos://%s/%s", testVersionBucketAlias, "single-small")
			args := []string{"cp", localFileName, cosFileName, "--disable-crc64"}
			cmd.SetArgs(args)
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})
		Convey("下载单个文件的指定版本", func() {
			clearCmd()
			cmd := rootCmd
			localFileName := fmt.Sprintf("%s/download/single-small", testDir)
			cosFileName := fmt.Sprintf("cos://%s/%s", testVersionBucketAlias, "single-small")
			opt := &cos.BucketGetObjectVersionsOptions{
				Prefix:          "single-small",
				Delimiter:       "",
				EncodingType:    "url",
				VersionIdMarker: "",
				KeyMarker:       "",
				MaxKeys:         0,
			}

			res, _, _ := c.Bucket.GetObjectVersions(context.Background(), opt)

			var versionId string
			for _, object := range res.Version {
				versionId = object.VersionId
				break
			}

			args := []string{"cp", cosFileName, localFileName, "--version-id", versionId}
			cmd.SetArgs(args)
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})
		Convey("跨桶拷贝单个文件的指定版本", func() {
			clearCmd()
			cmd := rootCmd
			srcPath := fmt.Sprintf("cos://%s/%s", testVersionBucketAlias, "single-small")
			dstPath := fmt.Sprintf("cos://%s/%s", testAlias1, "single-copy")
			opt := &cos.BucketGetObjectVersionsOptions{
				Prefix:          "single-small",
				Delimiter:       "",
				EncodingType:    "url",
				VersionIdMarker: "",
				KeyMarker:       "",
				MaxKeys:         0,
			}

			res, _, _ := c.Bucket.GetObjectVersions(context.Background(), opt)

			var versionId string
			for _, object := range res.Version {
				versionId = object.VersionId
				break
			}
			args := []string{"cp", srcPath, dstPath, "--version-id", versionId}
			cmd.SetArgs(args)
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})
		Convey("upload", func() {
			Convey("上传单个小文件并关闭crc64校验", func() {
				clearCmd()
				cmd := rootCmd
				localFileName := fmt.Sprintf("%s/small-file/0", testDir)
				cosFileName := fmt.Sprintf("cos://%s/%s", testAlias1, "single-small")
				args := []string{"cp", localFileName, cosFileName, "--disable-crc64"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("上传多个小文件", func() {
				clearCmd()
				cmd := rootCmd
				localFileName := fmt.Sprintf("%s/small-file", testDir)
				cosFileName := fmt.Sprintf("cos://%s/%s", testAlias1, "multi-small")
				args := []string{"cp", localFileName, cosFileName, "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("上传单个大文件", func() {
				clearCmd()
				cmd := rootCmd
				localFileName := fmt.Sprintf("%s/big-file/0", testDir)
				cosFileName := fmt.Sprintf("cos://%s/%s", testAlias1, "single-big")
				args := []string{"cp", localFileName, cosFileName}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("上传多个大文件", func() {
				clearCmd()
				cmd := rootCmd
				localFileName := fmt.Sprintf("%s/big-file", testDir)
				cosFileName := fmt.Sprintf("cos://%s/%s", testAlias1, "multi-big")
				args := []string{"cp", localFileName, cosFileName, "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})
		Convey("Copy", func() {
			Convey("桶内拷贝单个文件", func() {
				clearCmd()
				cmd := rootCmd
				srcPath := fmt.Sprintf("cos://%s/%s", testAlias1, "single-big")
				dstPath := fmt.Sprintf("cos://%s/%s", testAlias1, "single-copy")
				args := []string{"cp", srcPath, dstPath}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("桶内拷贝多个文件", func() {
				clearCmd()
				cmd := rootCmd
				srcPath := fmt.Sprintf("cos://%s/%s", testAlias1, "multi-big")
				dstPath := fmt.Sprintf("cos://%s/%s", testAlias1, "multi-copy")
				args := []string{"cp", srcPath, dstPath, "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("跨桶拷贝单个小文件", func() {
				clearCmd()
				cmd := rootCmd
				srcPath := fmt.Sprintf("cos://%s/%s", testAlias1, "single-small")
				dstPath := fmt.Sprintf("cos://%s/%s", testAlias2, "single-copy-small")
				args := []string{"cp", srcPath, dstPath}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("跨桶拷贝多个小文件", func() {
				clearCmd()
				cmd := rootCmd
				srcPath := fmt.Sprintf("cos://%s/%s", testAlias1, "multi-small")
				dstPath := fmt.Sprintf("cos://%s/%s", testAlias2, "multi-copy-small")
				args := []string{"cp", srcPath, dstPath, "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("跨桶拷贝单个大文件", func() {
				clearCmd()
				cmd := rootCmd
				srcPath := fmt.Sprintf("cos://%s/%s", testAlias1, "single-big")
				dstPath := fmt.Sprintf("cos://%s/%s", testAlias2, "single-copy-big")
				args := []string{"cp", srcPath, dstPath}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("跨桶拷贝多个大文件", func() {
				clearCmd()
				cmd := rootCmd
				srcPath := fmt.Sprintf("cos://%s/%s", testAlias1, "multi-big")
				dstPath := fmt.Sprintf("cos://%s/%s", testAlias2, "multi-copy-big")
				args := []string{"cp", srcPath, dstPath, "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})
		Convey("Download", func() {
			Convey("下载单个小文件", func() {
				clearCmd()
				cmd := rootCmd
				localFileName := fmt.Sprintf("%s/download/single-small", testDir)
				cosFileName := fmt.Sprintf("cos://%s/%s", testAlias2, "single-copy-small")
				args := []string{"cp", cosFileName, localFileName}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("下载多个小文件", func() {
				clearCmd()
				cmd := rootCmd
				localFileName := fmt.Sprintf("%s/download/small-file", testDir)
				cosFileName := fmt.Sprintf("cos://%s/%s", testAlias2, "multi-copy-small")
				args := []string{"cp", cosFileName, localFileName, "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("下载单个大文件", func() {
				clearCmd()
				cmd := rootCmd
				localFileName := fmt.Sprintf("%s/download/single-big", testDir)
				cosFileName := fmt.Sprintf("cos://%s/%s", testAlias2, "single-copy-big")
				args := []string{"cp", cosFileName, localFileName}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("下载多个大文件", func() {
				clearCmd()
				cmd := rootCmd
				localFileName := fmt.Sprintf("%s/download/big-file", testDir)
				cosFileName := fmt.Sprintf("cos://%s/%s", testAlias2, "multi-copy-big")
				args := []string{"cp", cosFileName, localFileName, "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})
		Convey("fail", func() {
			Convey("Not enough argument", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"cp"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("storageClass", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"cp", "cos://abc", "./test", "--storage-class", "STANDARD"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("MetaStringToHeader", func() {
				patches := ApplyFunc(util.MetaStringToHeader, func(string) (util.Meta, error) {
					return util.Meta{}, fmt.Errorf("test meta error")
				})
				defer patches.Reset()
				clearCmd()
				cmd := rootCmd
				args := []string{"cp", "cos://abc", "cos://abc"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("retryNum", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"cp", "cos://abc", "cos://abc", "--retry-num", "11"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("errRetryNum", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"cp", "cos://abc", "cos://abc", "--err-retry-num", "11"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("errRetryInterval", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"cp", "cos://abc", "cos://abc", "--err-retry-interval", "11"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("formatURL0", func() {
				patches := ApplyFunc(util.FormatUrl, func(urlStr string) (util.StorageUrl, error) {
					return nil, fmt.Errorf("test formatURL 0 error")
				})
				defer patches.Reset()
				clearCmd()
				cmd := rootCmd
				args := []string{"cp", "cos://abc", "cos://abc"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("formatURL1", func() {
				patches := ApplyFunc(util.FormatUrl, func(urlStr string) (util.StorageUrl, error) {
					if urlStr == "cos://abc" {
						return nil, nil
					} else {
						return nil, fmt.Errorf("test formatURL 1 error")
					}
				})
				defer patches.Reset()
				clearCmd()
				cmd := rootCmd
				args := []string{"cp", "cos://abc", "cos://123"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("tow local file", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"cp", "./abc", "./123"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("no -r but -i", func() {
				patches := ApplyFunc(util.GetFilter, func(string, string) (bool, []util.FilterOptionType) {
					tmp := []util.FilterOptionType{
						{},
					}
					return true, tmp
				})
				defer patches.Reset()
				clearCmd()
				cmd := rootCmd
				args := []string{"cp", "./abc", "cos://123", "--include", "abc"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("Upload", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"cp", "./abc", "cos://123", "--disable-crc64"}
				cmd.SetArgs(args)
				Convey("CheckPath", func() {
					patches := ApplyFunc(util.CheckPath, func(fileUrl util.StorageUrl, fo *util.FileOperations, pathType string) error {
						return fmt.Errorf("test CheckPath error")
					})
					defer patches.Reset()
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("NewClient", func() {
					patches := ApplyFunc(util.NewClient, func(config *util.Config, param *util.Param, bucketName string) (client *cos.Client, err error) {
						return nil, fmt.Errorf("test NewClient error")
					})
					defer patches.Reset()
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("FormatUploadPath", func() {
					patches := ApplyFunc(util.FormatUploadPath, func(fileUrl util.StorageUrl, cosUrl util.StorageUrl, fo *util.FileOperations) error {
						return fmt.Errorf("test FormatUploadPath error")
					})
					defer patches.Reset()
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
			})
			Convey("Download", func() {
				cosFileName := fmt.Sprintf("cos://%s/%s", testAlias2, "single-copy-small")
				clearCmd()
				cmd := rootCmd
				args := []string{"cp", cosFileName, "./abc", "--disable-crc64"}
				cmd.SetArgs(args)
				Convey("CheckPath", func() {
					patches := ApplyFunc(util.CheckPath, func(fileUrl util.StorageUrl, fo *util.FileOperations, pathType string) error {
						return fmt.Errorf("test CheckPath error")
					})
					defer patches.Reset()
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("NewClient", func() {
					patches := ApplyFunc(util.NewClient, func(config *util.Config, param *util.Param, bucketName string) (client *cos.Client, err error) {
						return nil, fmt.Errorf("test NewClient error")
					})
					defer patches.Reset()
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("Head", func() {
					var c *cos.BucketService
					patches := ApplyMethodFunc(reflect.TypeOf(c), "Head", func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
						return nil, fmt.Errorf("test Head error")
					})
					defer patches.Reset()
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("OFS", func() {
					patches := ApplyFunc(util.FormatDownloadPath, func(cosUrl util.StorageUrl, fileUrl util.StorageUrl, fo *util.FileOperations, c *cos.Client) error {
						return fmt.Errorf("test FormatDownloadPath error")
					})
					defer patches.Reset()
					var c http.Header
					patches.ApplyMethodFunc(c, "Get", func(key string) string {
						if key == "X-Cos-Bucket-Arch" {
							return "OFS"
						}
						return ""
					})
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("FormatDownloadPath", func() {
					patches := ApplyFunc(util.FormatDownloadPath, func(cosUrl util.StorageUrl, fileUrl util.StorageUrl, fo *util.FileOperations, c *cos.Client) error {
						return fmt.Errorf("test FormatDownloadPath error")
					})
					defer patches.Reset()
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("Download", func() {
					patches := ApplyFunc(util.Download, func(c *cos.Client, cosUrl util.StorageUrl, fileUrl util.StorageUrl, fo *util.FileOperations) error {
						return fmt.Errorf("test Download error")
					})
					defer patches.Reset()
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
			})
			Convey("CosCopy", func() {
				srcPath := fmt.Sprintf("cos://%s/%s", testAlias1, "single-big")
				dstPath := fmt.Sprintf("cos://%s/%s", testAlias1, "single-copy")
				clearCmd()
				cmd := rootCmd
				args := []string{"cp", srcPath, dstPath, "--disable-crc64"}
				cmd.SetArgs(args)
				Convey("NewClient src", func() {
					patches := ApplyFunc(util.NewClient, func(config *util.Config, param *util.Param, bucketName string) (client *cos.Client, err error) {
						return nil, fmt.Errorf("test NewClient src error")
					})
					defer patches.Reset()
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("NewClient dest", func() {
					index := false
					patches := ApplyFunc(util.NewClient, func(config *util.Config, param *util.Param, bucketName string) (client *cos.Client, err error) {
						if !index {
							index = true
							return nil, nil
						}
						return nil, fmt.Errorf("test NewClient dest error")
					})
					defer patches.Reset()
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("OFS", func() {
					patches := ApplyFunc(util.FormatCopyPath, func(srcUrl util.StorageUrl, destUrl util.StorageUrl, fo *util.FileOperations, srcClient *cos.Client) error {
						return fmt.Errorf("test FormatCopyPath error")
					})
					defer patches.Reset()
					var c http.Header
					patches.ApplyMethodFunc(c, "Get", func(key string) string {
						if key == "X-Cos-Bucket-Arch" {
							return "OFS"
						}
						return ""
					})
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("FormatCopyPath", func() {
					patches := ApplyFunc(util.FormatCopyPath, func(srcUrl util.StorageUrl, destUrl util.StorageUrl, fo *util.FileOperations, srcClient *cos.Client) error {
						return fmt.Errorf("test FormatCopyPath error")
					})
					defer patches.Reset()
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("CosCopy", func() {
					patches := ApplyFunc(util.CosCopy, func(srcClient *cos.Client, destClient *cos.Client, srcUrl util.StorageUrl, destUrl util.StorageUrl, fo *util.FileOperations) error {
						return fmt.Errorf("test CosCopy error")
					})
					defer patches.Reset()
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
			})
		})
	})
}
