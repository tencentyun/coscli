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

func TestMvCmd(t *testing.T) {
	fmt.Println("TestMvCmd")
	testBucket = randStr(8)
	testAlias = testBucket + "-alias"
	// testOfsBucket = randStr(8)
	// testOfsBucketAlias = testOfsBucket + "-alias"
	setUp(testBucket, testAlias, testEndpoint, false)
	defer tearDown(testBucket, testAlias, testEndpoint)
	// setUp(testOfsBucket, testOfsBucketAlias, testEndpoint, true)
	// defer tearDown(testOfsBucket, testOfsBucketAlias, testEndpoint)
	genDir(testDir, 3)
	defer delDir(testDir)
	localFileName := fmt.Sprintf("%s/small-file", testDir)
	cosFileName := fmt.Sprintf("cos://%s/%s", testAlias, "multi-small")
	// ofsFileName := fmt.Sprintf("cos://%s/%s", testOfsBucketAlias, "multi-small")
	clearCmd()
	cmd := rootCmd
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	args := []string{"cp", localFileName, cosFileName, "-r"}
	cmd.SetArgs(args)
	cmd.Execute()
	// args = []string{"cp", localFileName, ofsFileName, "-r"}
	// clearCmd()
	// cmd = rootCmd
	// cmd.SetArgs(args)
	// cmd.Execute()
	Convey("Test coscli mv", t, func() {
		Convey("success", func() {
			// Convey("ofs", func() {
			// 	clearCmd()
			// 	cmd := rootCmd
			// 	args := []string{"mv", fmt.Sprintf("%s/0", ofsFileName), fmt.Sprintf("cos://%s/0", testOfsBucketAlias)}
			// 	cmd.SetArgs(args)
			// 	e := cmd.Execute()
			// 	So(e, ShouldBeNil)
			// })
			Convey("not ofs", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"mv", fmt.Sprintf("%s/0", cosFileName), fmt.Sprintf("cos://%s/%s/0", testAlias, "testmv")}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("not ofs but -r", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"mv", cosFileName, fmt.Sprintf("cos://%s/%s", testAlias, "testmv"), "-r"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})
		Convey("fail", func() {
			Convey("not enough arguments", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"mv", "abc"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("storage-class", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"mv", "cos://abc", "cos://abc", "--storage-class", "STANDARD"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("meta", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(util.MetaStringToHeader, func(string) (util.Meta, error) {
					return util.Meta{}, fmt.Errorf("test meta error")
				})
				defer patches.Reset()
				args := []string{"mv", "cos://abc", "cos://abc", "--storage-class", ""}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("not cospath", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"mv", "~/.abc", "cos://abc"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("not equal cospath", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"mv", "cos://bcd", "cos://abc"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
		})
	})
}

func TestMove(t *testing.T) {
	testBucket = randStr(8)
	testAlias = testBucket + "-alias"
	testOfsBucket = randStr(8)
	testOfsBucketAlias = testOfsBucket + "-alias"
	cosFileName := fmt.Sprintf("cos://%s/%s", testAlias, "multi-small")
	cosOfsFileName := fmt.Sprintf("cos://%s/123", testOfsBucketAlias)
	setUp(testBucket, testAlias, testEndpoint, false)
	defer tearDown(testBucket, testAlias, testEndpoint)
	setUp(testOfsBucket, testOfsBucketAlias, testEndpoint, true)
	defer tearDown(testOfsBucket, testOfsBucketAlias, testEndpoint)

	Convey("Test func move", t, func() {
		Convey("NewClient", func() {
			patches := ApplyFunc(util.NewClient, func(config *util.Config, param *util.Param, bucketName string) (client *cos.Client, err error) {
				return nil, fmt.Errorf("test NewClient error")
			})
			defer patches.Reset()
			Convey("move", func() {
				args := []string{cosFileName, cosFileName}
				e := move(args, false, "", "", util.Meta{}, "")
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("MoveObjects", func() {
				args := []string{cosFileName, cosFileName}
				e := moveObjects(args, "", "")
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("MoveObject", func() {
				args := []string{cosFileName, cosFileName}
				e := moveObject(args)
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("recursivemoveObject", func() {
				e := recursivemoveObject(testAlias, "")
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
		})
		Convey("Head", func() {
			var c *cos.BucketService
			patches := ApplyMethodFunc(reflect.TypeOf(c), "Head", func(ctx context.Context, opt ...*cos.BucketHeadOptions) (*cos.Response, error) {
				return nil, fmt.Errorf("test Head error")
			})
			defer patches.Reset()
			args := []string{cosFileName, cosFileName}
			e := move(args, false, "", "", util.Meta{}, "")
			fmt.Printf(" : %v", e)
			So(e, ShouldBeError)
		})
		Convey("OFS", func() {
			patches := ApplyFunc(util.PutRename, func(ctx context.Context, config *util.Config, param *util.Param, c *cos.Client, name string, dstURL string, closeBody bool) (resp *http.Response, err error) {
				return nil, fmt.Errorf("test PutRename error")
			})
			defer patches.Reset()
			clearCmd()
			cmd := rootCmd
			args := []string{"mv", cosOfsFileName, cosOfsFileName}
			cmd.SetArgs(args)
			e := cmd.Execute()
			fmt.Printf(" : %v", e)
			So(e, ShouldBeError)
		})
		Convey("not OFS", func() {
			Convey("cosCopy", func() {
				patches := ApplyFunc(cosCopy, func(args []string, recursive bool, include string, exclude string, meta util.Meta, storageClass string) error {
					return fmt.Errorf("test cosCopy error")
				})
				defer patches.Reset()
				args := []string{cosFileName, cosFileName}
				e := move(args, true, "", "", util.Meta{}, "")
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			// Convey("moveObject", func() {
			// 	patches := ApplyFunc(cosCopy, func(args []string, recursive bool.., include string, exclude string, meta util.Meta, storageClass string) error {
			// 		return nil
			// 	})
			// 	defer patches.Reset()
			// 	args := []string{cosFileName, cosFileName}
			// 	e := move(args, false, "", "", util.Meta{}, "")
			// 	fmt.Printf(" : %v", e)
			// 	So(e, ShouldBeError)
			// })
		})

		Convey("MoveObjects", func() {
			Convey("GetObjectsListIterator", func() {
				patches := ApplyFunc(util.GetObjectsListIterator, func(c *cos.Client, prefix string, marker string, include string, exclude string) (objects []cos.Object, isTruncated bool, nextMarker string, commonPrefixes []string, err error) {
					return nil, false, "", nil, fmt.Errorf("test GetObjectsListIterator error")
				})
				defer patches.Reset()
				args := []string{cosFileName, cosFileName}
				e := moveObjects(args, "", "")
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("commonPrefixes", func() {
				patches := ApplyFunc(util.GetObjectsListIterator, func(c *cos.Client, prefix string, marker string, include string, exclude string) (objects []cos.Object, isTruncated bool, nextMarker string, commonPrefixes []string, err error) {
					return nil, false, "", []string{"1"}, nil
				})
				defer patches.Reset()
				patches.ApplyFunc(getFilesAndDirs, func(c *cos.Client, cosDir string, nextMarker string, include string, exclude string) (files []string, err error) {
					return nil, fmt.Errorf("test getFilesAndDirs error")
				})
				args := []string{cosFileName, cosFileName}
				e := moveObjects(args, "", "")
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("recursivemoveObject", func() {
				patches := ApplyFunc(util.GetObjectsListIterator, func(c *cos.Client, prefix string, marker string, include string, exclude string) (objects []cos.Object, isTruncated bool, nextMarker string, commonPrefixes []string, err error) {
					return nil, false, "", []string{"1"}, nil
				})
				defer patches.Reset()
				patches.ApplyFunc(getFilesAndDirs, func(c *cos.Client, cosDir string, nextMarker string, include string, exclude string) (files []string, err error) {
					return []string{"1"}, nil
				})
				patches.ApplyFunc(recursivemoveObject, func(bucketName string, cosPath string) error {
					return fmt.Errorf("test recursivemoveObject error")
				})
				args := []string{cosFileName, cosFileName}
				e := moveObjects(args, "", "")
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("DeleteMulti", func() {
				patches := ApplyFunc(util.GetObjectsListIterator, func(c *cos.Client, prefix string, marker string, include string, exclude string) (objects []cos.Object, isTruncated bool, nextMarker string, commonPrefixes []string, err error) {
					return nil, false, "", []string{}, nil
				})
				defer patches.Reset()
				var c *cos.ObjectService
				patches.ApplyMethodFunc(c, "DeleteMulti", func(ctx context.Context, opt *cos.ObjectDeleteMultiOptions) (*cos.ObjectDeleteMultiResult, *cos.Response, error) {
					return nil, nil, fmt.Errorf("test DeleteMulti error")
				})
				args := []string{cosFileName, cosFileName}
				e := moveObjects(args, "", "")
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
		})
		Convey("moveObject", func() {
			var c *cos.ObjectService
			patches := ApplyMethodFunc(c, "Delete", func(ctx context.Context, name string, opt ...*cos.ObjectDeleteOptions) (*cos.Response, error) {
				return nil, fmt.Errorf("test Delete error")
			})
			defer patches.Reset()
			args := []string{cosFileName, cosFileName}
			e := moveObject(args)
			fmt.Printf(" : %v", e)
			So(e, ShouldBeError)
		})
		Convey("recursivemoveObject", func() {
			var c *cos.ObjectService
			patches := ApplyMethodFunc(c, "Delete", func(ctx context.Context, name string, opt ...*cos.ObjectDeleteOptions) (*cos.Response, error) {
				return nil, fmt.Errorf("test Delete error")
			})
			defer patches.Reset()
			e := recursivemoveObject(testAlias, "")
			fmt.Printf(" : %v", e)
			So(e, ShouldBeError)
		})
	})
}
