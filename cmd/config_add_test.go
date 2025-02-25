package cmd

import (
	"fmt"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/viper"
)

func TestConfigAddCmd(t *testing.T) {
	fmt.Println("TestConfigAddCmd")
	testBucket = randStr(8)
	testAlias = testBucket + "-alias"
	setUp(testBucket, testAlias, testEndpoint, false, false)
	defer tearDown(testBucket, testAlias, testEndpoint, false)
	copyYaml()
	defer delYaml()
	clearCmd()
	cmd := rootCmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	Convey("Test coscil config add", t, func() {
		// 成功不需要测，setup里就用过了
		// Convey("success", func() {
		// 	Convey("All have", func() {
		// 		cmd := rootCmd
		// 		args := []string{"config", "add", "-b",
		// 			fmt.Sprintf("%s-%s", testBucket, appID), "-e", testEndpoint, "-a", testAlias}
		// 		cmd.SetArgs(args)
		// 		e := cmd.Execute()
		// 		So(e, ShouldBeNil)
		// 	})
		// })
		Convey("fail", func() {
			Convey("Bucket already exist: name", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"config", "add", "-b",
					fmt.Sprintf("%s-%s", testBucket, appID), "-e", testEndpoint, "-a", "testAlias"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("Bucket already exist: alias-name", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"config", "add", "-b",
					fmt.Sprintf("%s-%s", "testBucket", appID), "-e", testEndpoint, "-a", fmt.Sprintf("%s-%s", testBucket, appID)}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("Bucket already exist: alias", func() {
				clearCmd()
				cmd := rootCmd
				args := []string{"config", "add", "-b",
					fmt.Sprintf("%s-%s", "testBucket", appID), "-e", testEndpoint, "-a", testAlias}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("WriteConfigAs", func() {
				patches := ApplyFunc(viper.WriteConfigAs, func(string) error {
					config.Buckets = config.Buckets[:len(config.Buckets)-1]
					viper.Set("cos.buckets", config.Buckets)
					return fmt.Errorf("test write configas error")
				})
				defer patches.Reset()
				Convey("cfgFile[0]!=~", func() {
					clearCmd()
					cmd := rootCmd
					args := []string{"config", "add", "-b",
						fmt.Sprintf("%s-%s", "testBucket", appID), "-e", testEndpoint, "-a", "testAlias", "-c", "./test.yaml"}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
				Convey("no cfgFile", func() {
					clearCmd()
					cmd := rootCmd
					args := []string{"config", "add", "-b",
						fmt.Sprintf("%s-%s", "testBucket", appID), "-e", testEndpoint, "-a", ""}
					cmd.SetArgs(args)
					e := cmd.Execute()
					fmt.Printf(" : %v", e)
					So(e, ShouldBeError)
				})
			})
		})
	})
}
