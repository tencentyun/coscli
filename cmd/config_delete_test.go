package cmd

import (
	"coscli/util"
	"fmt"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/viper"
)

func TestConfigDeleteCmd(t *testing.T) {
	fmt.Println("TestConfigDeleteCmd")
	// 恢复原来的 Buckets
	clearCmd()
	cmd := rootCmd
	buckets := config.Buckets
	defer func() {
		viper.Set("cos.buckets", buckets)
		viper.WriteConfigAs(viper.ConfigFileUsed())
	}()
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	Convey("Test coscil config delete", t, func() {
		// Convey("success", func() {
		// 	args := []string{"config", "delete", "-a",
		// 		fmt.Sprintf("%s-%s", testBucket, appID)}
		// 	cmd.SetArgs(args)
		// 	e := cmd.Execute()
		// 	So(e, ShouldBeNil)
		// })
		Convey("fail", func() {
			Convey("FindBucket", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(util.FindBucket, func(config *util.Config, bucketName string) (util.Bucket, int, error) {
					return util.Bucket{}, 0, fmt.Errorf("test findbucket fail")
				})
				defer patches.Reset()
				args := []string{"config", "delete", "-a", "testAlias"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("FindBucket i<0", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(util.FindBucket, func(config *util.Config, bucketName string) (util.Bucket, int, error) {
					return util.Bucket{}, -1, nil
				})
				defer patches.Reset()
				args := []string{"config", "delete", "-a", "testAlias"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
			Convey("viper.WriteConfigAs", func() {
				clearCmd()
				cmd := rootCmd
				patches := ApplyFunc(viper.WriteConfigAs, func(string) error {
					return fmt.Errorf("test WriteConfigAs fail")
				})
				defer patches.Reset()
				patches.ApplyFunc(util.FindBucket, func(config *util.Config, bucketName string) (util.Bucket, int, error) {
					return util.Bucket{}, 0, nil
				})
				args := []string{"config", "delete", "-a", "testAlias"}
				cmd.SetArgs(args)
				e := cmd.Execute()
				fmt.Printf(" : %v", e)
				So(e, ShouldBeError)
			})
		})
	})
}
