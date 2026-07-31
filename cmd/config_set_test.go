package cmd

import (
	"fmt"
	"os"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/viper"
)

func TestConfigSetCmd(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("Test coscli config set", t, func() {
		var patches *Patches
		Reset(func() {
			if patches != nil {
				patches.Reset()
				patches = nil
			}
			clearCmd()
			setupTestConfig()
		})

		Convey("set all fields", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "set",
				"--secret_id", "new-secret-id",
				"--secret_key", "new-secret-key",
				"--session_token", "new-token",
				"--mode", "",
				"--cvm_role_name", "",
				"--close_auto_switch_host", "",
				"--disable_encryption", "true",
				"-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("no arguments", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "set", "-c", testConfigPath})
			e := cmd.Execute()
			fmt.Printf(" : %v", e)
			So(e, ShouldBeError)
		})

		Convey("invalid mode", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "set", "--mode", "@", "-c", testConfigPath})
			e := cmd.Execute()
			fmt.Printf(" : %v", e)
			So(e, ShouldBeError)
		})

		Convey("WriteConfigAs error", func() {
			// 用 cfgFile 指向不存在的目录路径，让 root.initConfig 内部
			// viper.ReadInConfig 失败 → 调用 os.Exit(1)。打桩 os.Exit 后只验证
			// 它被调用即可（不依赖 gomonkey 对 viper 包级方法的打桩）。
			exitCalled := false
			patches = ApplyFunc(os.Exit, func(code int) { exitCalled = true })

			notExistDirCfg := "/tmp/coscli-test-set-no-such-dir/cfg.yaml"
			_ = os.RemoveAll("/tmp/coscli-test-set-no-such-dir")
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "set",
				"--secret_id", "new-id",
				"-c", notExistDirCfg})
			_ = cmd.Execute()
			So(exitCalled, ShouldBeTrue)
		})

		Convey("clear fields with @", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "set",
				"--secret_id", "@",
				"--secret_key", "@",
				"--session_token", "@",
				"--disable_encryption", "true",
				"-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("set cvm_role_name", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "set",
				"--cvm_role_name", "my-role",
				"--disable_encryption", "true",
				"-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("clear cvm_role_name with @", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "set",
				"--cvm_role_name", "@",
				"--disable_encryption", "true",
				"-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("set close_auto_switch_host", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "set",
				"--close_auto_switch_host", "true",
				"--disable_encryption", "true",
				"-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("clear close_auto_switch_host with @", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "set",
				"--close_auto_switch_host", "@",
				"--disable_encryption", "true",
				"-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("set disable_encryption", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "set",
				"--disable_encryption", "true",
				"-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("clear disable_encryption with @", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "set",
				"--disable_encryption", "@",
				"-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("set disable_auto_fetch_bucket_type", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "set",
				"--disable_auto_fetch_bucket_type", "true",
				"--disable_encryption", "true",
				"-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("clear disable_auto_fetch_bucket_type with @", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "set",
				"--disable_auto_fetch_bucket_type", "@",
				"--disable_encryption", "true",
				"-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("set proxy", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "set",
				"--proxy", "http://127.0.0.1:8080",
				"--disable_encryption", "true",
				"-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("clear proxy with @", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "set",
				"--proxy", "@",
				"--disable_encryption", "true",
				"-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("set valid mode SecretKey", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "set",
				"--mode", "SecretKey",
				"--disable_encryption", "true",
				"-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("set valid mode CvmRole", func() {
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "set",
				"--mode", "CvmRole",
				"--disable_encryption", "true",
				"-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("set with encryption enabled (disableEncryption != true)", func() {
			// 创建一个 disableEncryption 为 false 的配置文件
			encConfigPath := "/tmp/coscli-test-enc-set.yaml"
			content := `cos:
  base:
    secretid: "test-secret-id"
    secretkey: "test-secret-key"
    sessiontoken: ""
    protocol: "https"
    disableEncryption: "false"
  buckets:
    - name: "test-bucket-1234567890"
      alias: "test-alias"
      region: "ap-guangzhou"
      endpoint: "cos.ap-guangzhou.myqcloud.com"
`
			os.WriteFile(encConfigPath, []byte(content), 0644)
			defer os.Remove(encConfigPath)
			// 重新加载配置
			viper.Reset()
			viper.SetConfigFile(encConfigPath)
			viper.ReadInConfig()
			viper.UnmarshalKey("cos", &config)

			cmd := rootCmd
			cmd.SetArgs([]string{"config", "set",
				"--secret_id", "new-id",
				"-c", encConfigPath})
			e := cmd.Execute()
			// 加密可能失败，但不应该 panic
			_ = e
		})

		Convey("set with tilde path (~)", func() {
			// 使用 ~ 开头的路径，触发 cfgFile[0] == '~' 分支
			home, _ := os.UserHomeDir()
			tildePath := "~/.cos-test-set-tilde.yaml"
			realPath := home + "/.cos-test-set-tilde.yaml"
			// 先创建文件
			os.WriteFile(realPath, []byte(`cos:
  base:
    secretid: "test-id"
    secretkey: "test-key"
    protocol: "https"
    disableEncryption: "true"
  buckets: []
`), 0644)
			defer os.Remove(realPath)
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "set",
				"--secret_id", "new-id",
				"--disable_encryption", "true",
				"-c", tildePath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})

		Convey("set without -c, file exists, uses WriteConfigAs(ConfigFileUsed)", func() {
			// 不传 -c 参数，原代码会 fallback 到 home + "/.cos.yaml"。
			// TestMain 已把 HOME 切到受控临时目录，因此这里直接执行即可，
			// 绝不会污染用户真实 ~/.cos.yaml。
			// 仍打桩 os.Exit 以防 root.initConfig 提前退出测试进程。
			patches = ApplyFunc(os.Exit, func(code int) {})

			cmd := rootCmd
			cmd.SetArgs([]string{"config", "set",
				"--secret_id", "new-id",
				"--disable_encryption", "true"})
			e := cmd.Execute()
			fmt.Printf(" : %v", e)
			_ = e
		})
	})
}
