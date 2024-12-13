package cmd

import (
	"coscli/util"
	"fmt"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestBucket_versionCmd(t *testing.T) {
	fmt.Println("TestBucket_versionCmd")
	testBucket = randStr(8)
	testAlias = testBucket + "-alias"
	setUp(testBucket, testAlias, testEndpoint, false)
	defer tearDown(testBucket, testAlias, testEndpoint)
	clearCmd()
	cmd := rootCmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	Convey("test coscli bucket_versioning", t, func() {
		Convey("success", func() {
			Convey("put", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"bucket-versioning", "--method", "put",
					fmt.Sprintf("cos://%s", testAlias), "Enabled"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("get", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"bucket-versioning", "--method", "get",
					fmt.Sprintf("cos://%s", testAlias)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})
		Convey("fail", func() {
			Convey("put", func() {
				Convey("not enough arguments", func() {
					clearCmd()
					cmd := rootCmd
					args := []string{"bucket-versioning", "--method", "put",
						fmt.Sprintf("cos://%s", testAlias)}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("clinet err", func() {
					clearCmd()
					cmd := rootCmd
					patches := ApplyFunc(util.NewClient, func(config *util.Config, param *util.Param, bucketName string) (client *cos.Client, err error) {
						return nil, fmt.Errorf("test put client error")
					})
					defer patches.Reset()
					args := []string{"bucket-versioning", "--method", "put",
						fmt.Sprintf("cos://%s", testAlias), "testval"}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("invalid status", func() {
					clearCmd()
					cmd := rootCmd
					args := []string{"bucket-versioning", "--method", "put",
						fmt.Sprintf("cos://%s", testAlias), "testStatus"}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				//Convey("PutVersioning failed", func() {
				//	clearCmd()
				//	cmd := rootCmd
				//	args := []string{"bucket-versioning", "--method", "put",
				//		fmt.Sprintf("cos://%s", testAlias), "qcs:1#testval"}
				//	cmd.SetArgs(args)
				//	e := cmd.Execute()
				//	fmt.Printf(" : %v", e)
				//	So(e, ShouldBeError)
				//})
			})
			Convey("get", func() {
				Convey("clinet err", func() {
					clearCmd()
					cmd := rootCmd
					patches := ApplyFunc(util.NewClient, func(config *util.Config, param *util.Param, bucketName string) (client *cos.Client, err error) {
						return nil, fmt.Errorf("test get client error")
					})
					defer patches.Reset()
					args := []string{"bucket-versioning", "--method", "get",
						fmt.Sprintf("cos://%s", testAlias)}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
			})
		})
	})
}
