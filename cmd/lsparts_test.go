package cmd

import (
	"coscli/util"
	"fmt"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestLspartsCmd(t *testing.T) {
	fmt.Println("TestLspartsCmd")
	testBucket = randStr(8)
	testAlias = testBucket + "-alias"
	setUp(testBucket, testAlias, testEndpoint, false, false)
	defer tearDown(testBucket, testAlias, testEndpoint, false)
	clearCmd()
	cmd := rootCmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	Convey("Test coscli lsparts", t, func() {
		Convey("success", func() {
			Convey("ls uploads", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"lsparts", fmt.Sprintf("cos://%s", testAlias)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("ls parts", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(util.CheckUploadExist, func(c *cos.Client, cosUrl util.StorageUrl, uploadId string) (exist bool, err error) {
					return true, nil
				})
				defer patches.Reset()

				lsPatches := ApplyFunc(util.GetPartsListForLs, func(c *cos.Client, cosUrl util.StorageUrl, uploadId, partNumberMarker string, limit int) (err error, parts []cos.Object, isTruncated bool, nextPartNumberMarker string) {
					return nil, []cos.Object{
						{
							Key:          "123",
							PartNumber:   1,
							LastModified: "2024-12-17T08:34:48.000Z",
							ETag:         "58f06dd588d8ffb3beb46ada6309436b",
							Size:         33554432,
						},
					}, false, ""
				})
				defer lsPatches.Reset()

				args := []string{"lsparts", fmt.Sprintf("cos://%s", testAlias), "--upload-id", "123"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})

		})
		Convey("fail", func() {
			Convey("limit invalid", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"lsparts", fmt.Sprintf("cos://%s", testAlias), "--limit", "-1"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("New Client", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(util.NewClient, func(config *util.Config, param *util.Param, bucketName string) (client *cos.Client, err error) {
					return nil, fmt.Errorf("test formaturl error")
				})
				defer patches.Reset()
				args := []string{"lsparts", fmt.Sprintf("cos://%s", testAlias)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("GetUploadsListForLs", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(util.GetUploadsListForLs, func(c *cos.Client, cosUrl util.StorageUrl, uploadIDMarker, keyMarker string, limit int, recursive bool) (err error, uploads []struct {
					Key          string
					UploadID     string `xml:"UploadId"`
					StorageClass string
					Initiator    *cos.Initiator
					Owner        *cos.Owner
					Initiated    string
				}, isTruncated bool, nextUploadIDMarker, nextKeyMarker string) {
					return fmt.Errorf("test GetUpload client error"), nil, false, "", ""
				})
				defer patches.Reset()
				args := []string{"lsparts", fmt.Sprintf("cos://%s", testAlias)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("range uploads", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(util.GetUploadsListForLs, func(c *cos.Client, cosUrl util.StorageUrl, uploadIDMarker, keyMarker string, limit int, recursive bool) (err error, uploads []struct {
					Key          string
					UploadID     string `xml:"UploadId"`
					StorageClass string
					Initiator    *cos.Initiator
					Owner        *cos.Owner
					Initiated    string
				}, isTruncated bool, nextUploadIDMarker, nextKeyMarker string) {
					tmp := []struct {
						Key          string
						UploadID     string `xml:"UploadId"`
						StorageClass string
						Initiator    *cos.Initiator
						Owner        *cos.Owner
						Initiated    string
					}{
						{
							Key:      "666",
							UploadID: "888",
						},
					}

					return nil, tmp, false, "", ""
				})
				defer patches.Reset()
				args := []string{"lsparts", fmt.Sprintf("cos://%s", testAlias)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("upload not exist", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(util.CheckUploadExist, func(c *cos.Client, cosUrl util.StorageUrl, uploadId string) (exist bool, err error) {
					return false, nil
				})
				defer patches.Reset()

				args := []string{"lsparts", fmt.Sprintf("cos://%s", testAlias), "--upload-id", "1"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("ls parts error", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(util.CheckUploadExist, func(c *cos.Client, cosUrl util.StorageUrl, uploadId string) (exist bool, err error) {
					return true, nil
				})
				defer patches.Reset()

				lsPatches := ApplyFunc(util.GetPartsListForLs, func(c *cos.Client, cosUrl util.StorageUrl, uploadId, partNumberMarker string, limit int) (err error, parts []cos.Object, isTruncated bool, nextPartNumberMarker string) {
					return fmt.Errorf("test GetUpload client error"), nil, false, ""
				})
				defer lsPatches.Reset()

				args := []string{"lsparts", fmt.Sprintf("cos://%s", testAlias), "--upload-id", "1734424486bf8693045e9e926aa85008e3e58ddd5794fa68e0300d62d663c939e1a3b896d7"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
		})
	})
}
