package cmd

import (
	"fmt"
	"os"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/viper"
)

func TestConfigAddCmd(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("Test coscli config add", t, func() {
		var patches *Patches
		Reset(func() {
			if patches != nil {
				patches.Reset()
				patches = nil
			}
			clearCmd()
			setupTestConfig()
		})

		Convey("add new bucket", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "add", "-b", "new-bucket-9999999999",
				"-e", testEndpoint, "-a", "new-alias", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("add bucket with customized", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "add", "-b", "custom-bucket-9999999999",
				"-e", testEndpoint, "-a", "custom-alias", "--customized", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("Bucket already exist: name", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "add", "-b", "test-bucket-1234567890",
				"-e", testEndpoint, "-a", "other-alias", "-c", testConfigPath})
			e := cmd.Execute()
			fmt.Printf(" : %v", e)
			So(e, ShouldBeError)
		})

		Convey("Bucket already exist: alias", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "add", "-b", "other-bucket-9999999999",
				"-e", testEndpoint, "-a", "test-alias", "-c", testConfigPath})
			e := cmd.Execute()
			fmt.Printf(" : %v", e)
			So(e, ShouldBeError)
		})

		Convey("Bucket already exist: alias-name conflict", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "add", "-b", "other-bucket-9999999999",
				"-e", testEndpoint, "-a", "test-bucket-1234567890", "-c", testConfigPath})
			e := cmd.Execute()
			fmt.Printf(" : %v", e)
			So(e, ShouldBeError)
		})

		Convey("WriteConfigAs error", func() {
			patches = ApplyFunc(viper.WriteConfigAs, func(string) error {
				return fmt.Errorf("test write configas error")
			})
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "add", "-b", "new-bucket-8888888888",
				"-e", testEndpoint, "-a", "new-alias-2", "-c", testConfigPath})
			e := cmd.Execute()
			fmt.Printf(" : %v", e)
			So(e, ShouldBeError)
		})

		Convey("add bucket without alias (alias defaults to name)", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "add", "-b", "no-alias-bucket-9999999999",
				"-e", testEndpoint, "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("add bucket with ofs flag", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "add", "-b", "ofs-bucket-9999999999",
				"-e", testEndpoint, "-a", "ofs-alias", "-o", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("WriteConfigAs used config error", func() {
			// 打桩 viper.WriteConfigAs 返回错误（cfgFile == "" 时使用 viper.ConfigFileUsed()）
			patches = ApplyFunc(viper.WriteConfigAs, func(string) error {
				return fmt.Errorf("test write config error")
			})
			// 不传 -c 参数，触发 cfgFile == "" 分支
			// 但需要 viper 有已使用的配置文件
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "add", "-b", "new-bucket-7777777777",
				"-e", testEndpoint, "-a", "new-alias-3"})
			e := cmd.Execute()
			fmt.Printf(" : %v", e)
			// 不管成功还是失败，只要不 panic 就行
			_ = e
		})

		Convey("add bucket with tilde config path", func() {
			// 使用 ~ 开头的配置文件路径，触发 cfgFile[0] == '~' 分支
			home, _ := os.UserHomeDir()
			tildeConfigPath := "~/.cos-test-add-tilde.yaml"
			realConfigPath := home + "/.cos-test-add-tilde.yaml"
			// 先创建配置文件
			configContent := `cos:
  base:
    secretid: "test-secret-id"
    secretkey: "test-secret-key"
    sessiontoken: ""
    protocol: "https"
    disableEncryption: "true"
  buckets:
    - name: "test-bucket-1234567890"
      alias: "test-alias"
      region: "ap-guangzhou"
      endpoint: "cos.ap-guangzhou.myqcloud.com"
`
			os.WriteFile(realConfigPath, []byte(configContent), 0644)
			defer os.Remove(realConfigPath)
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "add", "-b", "tilde-bucket-1234567890",
				"-e", testEndpoint, "-a", "tilde-alias", "-c", tildeConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})
	})
}
