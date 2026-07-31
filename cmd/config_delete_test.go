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
	setupTestConfig()
	defer teardownTestConfig()

	Convey("Test coscli config delete", t, func() {
		var patches *Patches
		Reset(func() {
			if patches != nil {
				patches.Reset()
				patches = nil
			}
			clearCmd()
			setupTestConfig()
		})

		Convey("delete existing bucket", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "delete", "-a", "test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("FindBucket error", func() {
			patches = ApplyFunc(util.FindBucket, func(config *util.Config, bucketName string) (util.Bucket, int, error) {
				return util.Bucket{}, 0, fmt.Errorf("test findbucket fail")
			})
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "delete", "-a", "testAlias", "-c", testConfigPath})
			e := cmd.Execute()
			fmt.Printf(" : %v", e)
			So(e, ShouldBeError)
		})

		Convey("FindBucket index < 0", func() {
			patches = ApplyFunc(util.FindBucket, func(config *util.Config, bucketName string) (util.Bucket, int, error) {
				return util.Bucket{}, -1, nil
			})
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "delete", "-a", "testAlias", "-c", testConfigPath})
			e := cmd.Execute()
			fmt.Printf(" : %v", e)
			So(e, ShouldBeError)
		})

		Convey("WriteConfigAs error", func() {
			patches = ApplyFunc(viper.WriteConfigAs, func(string) error {
				return fmt.Errorf("test WriteConfigAs fail")
			})
			patches.ApplyFunc(util.FindBucket, func(config *util.Config, bucketName string) (util.Bucket, int, error) {
				return util.Bucket{}, 0, nil
			})
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "delete", "-a", "testAlias", "-c", testConfigPath})
			e := cmd.Execute()
			fmt.Printf(" : %v", e)
			So(e, ShouldBeError)
		})
	})
}
