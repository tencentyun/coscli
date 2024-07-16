package cmd

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestBucket_taggingCmd(t *testing.T) {
	fmt.Println("TestBucket_taggingCmd")
	setUp(testBucket, testAlias, testEndpoint)
	defer tearDown(testBucket, testAlias, testEndpoint)
	Convey("test coscli bucket_tagging", t, func() {
		Convey("success", func() {
			Convey("put", func() {
				cmd := rootCmd
				args := []string{"bucket-tagging", "--method", "put",
					fmt.Sprintf("cos://%s", testAlias), "testkey#testval"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("get", func() {
				cmd := rootCmd
				args := []string{"bucket-tagging", "--method", "get",
					fmt.Sprintf("cos://%s", testAlias)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
			Convey("delete", func() {
				cmd := rootCmd
				args := []string{"bucket-tagging", "--method", "delete",
					fmt.Sprintf("cos://%s", testAlias)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				So(e, ShouldBeNil)
			})
		})
		// Convey("fail", func() {
		// 	Convey("put", func() {
		// 		Convey("not enough arguments", func() {
		// 			cmd := rootCmd
		// 			args := []string{"bucket-tagging", "--method", "put",
		// 				fmt.Sprintf("cos://%s", testAlias)}
		// 			cmd.SetArgs(args)
		// 			e := cmd.Execute()
		// 			fmt.Println(cmd.OutOrStdout())
		// 			So(e, ShouldBeError)
		// 		})
		// 		Convey("invalid tag", func() {
		// 			cmd := rootCmd
		// 			args := []string{"bucket-tagging", "--method", "put",
		// 				fmt.Sprintf("cos://%s", testAlias), "testval"}
		// 			cmd.SetArgs(args)
		// 			e := cmd.Execute()
		// 			fmt.Println(cmd.OutOrStdout())
		// 			So(e, ShouldBeError)
		// 		})
		// 		Convey("PutTagging failed", func() {
		// 			cmd := rootCmd
		// 			args := []string{"bucket-tagging", "--method", "put",
		// 				fmt.Sprintf("cos://%s", testAlias), "qcs:1#testval"}
		// 			cmd.SetArgs(args)
		// 			e := cmd.Execute()
		// 			fmt.Println(cmd.OutOrStdout())
		// 			So(e, ShouldBeError)
		// 		})
		// 	})
		// 	Convey("get", func() {
		// 		Convey("not enough arguments", func() {
		// 			cmd := rootCmd
		// 			args := []string{"bucket-tagging", "--method", "get"}
		// 			cmd.SetArgs(args)
		// 			e := cmd.Execute()
		// 			fmt.Println(cmd.OutOrStdout())
		// 			So(e, ShouldBeError)
		// 		})
		// 	})
		// 	Convey("delete", func() {
		// 		Convey("not enough arguments", func() {
		// 			cmd := rootCmd
		// 			args := []string{"bucket-tagging", "--method", "delete"}
		// 			cmd.SetArgs(args)
		// 			e := cmd.Execute()
		// 			fmt.Println(cmd.OutOrStdout())
		// 			So(e, ShouldBeError)
		// 		})
		// 	})
		// })
	})
}
