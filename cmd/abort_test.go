package cmd

import (
	"context"
	"coscli/util"
	"fmt"
	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tencentyun/cos-go-sdk-v5"
	"reflect"
	"testing"
)

func TestAbortCmd(t *testing.T) {
	fmt.Println("TestAbortCmd")
	testBucket = randStr(8)
	testAlias = testBucket + "-alias"
	setUp(testBucket, testAlias, testEndpoint, false, false)
	defer tearDown(testBucket, testAlias, testEndpoint, false)
	clearCmd()
	cmd := rootCmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	Convey("Test coscli abort", t, func() {
		Convey("success", func() {
			Convey("0 success 0 fail", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"abort",
					fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("1 success", func() {
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
				var c *cos.ObjectService
				patches.ApplyMethodFunc(reflect.TypeOf(c), "AbortMultipartUpload", func(ctx context.Context, name string, uploadID string, opt ...*cos.AbortMultipartUploadOptions) (*cos.Response, error) {
					return nil, nil
				})
				args := []string{"abort",
					fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("1 fail", func() {
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
				var c *cos.ObjectService
				patches.ApplyMethodFunc(reflect.TypeOf(c), "AbortMultipartUpload", func(ctx context.Context, name string, uploadID string, opt ...*cos.AbortMultipartUploadOptions) (*cos.Response, error) {
					return nil, fmt.Errorf("test abort fail")
				})

				args := []string{"abort",
					fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})
		Convey("failed", func() {
			Convey("not enough argument", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"abort"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("client fail", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(util.NewClient, func(config *util.Config, param *util.Param, bucketName string) (client *cos.Client, err error) {
					return nil, fmt.Errorf("test abort client error")
				})
				defer patches.Reset()
				args := []string{"abort",
					fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("GetUpload fail", func() {
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
				args := []string{"abort",
					fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
		})
	})
}
