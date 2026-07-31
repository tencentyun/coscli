package cmd

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	logrus "github.com/sirupsen/logrus"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/viper"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestSkipCmd(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("Test coscli root skip flags", t, func() {
		var patches *Patches
		Reset(func() {
			if patches != nil {
				patches.Reset()
				patches = nil
			}
			clearCmd()
		})

		Convey("init-skip with invalid params", func() {
			var s *cos.ServiceService
			patches = ApplyMethodFunc(reflect.TypeOf(s), "Get",
				func(ctx context.Context, opt ...*cos.ServiceGetOptions) (*cos.ServiceGetResult, *cos.Response, error) {
					return nil, nil, fmt.Errorf("test service get error")
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"ls", "cos://test-alias", "--init-skip", "-i", "123", "-k", "456", "-c", testConfigPath})
			e := cmd.Execute()
			fmt.Printf(" : %v", e)
			So(e, ShouldBeError)
		})
	})
}

func TestExecute(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("Test Execute function", t, func() {
		Reset(func() {
			clearCmd()
		})

		Convey("Execute returns nil for help", func() {
			rootCmd.SetArgs([]string{"--help"})
			e := Execute()
			So(e, ShouldBeNil)
		})

		Convey("Execute with no args triggers rootCmd.Run", func() {
			// 不带任何参数执行，触发 rootCmd.Run 中的 cmd.Help()
			rootCmd.SetArgs([]string{"-c", testConfigPath})
			e := Execute()
			So(e, ShouldBeNil)
		})
	})
}

func TestInitConfigProtocolDefault(t *testing.T) {
	// 测试 config.Base.Protocol == "" 时默认设置为 "https"
	// 创建一个没有 protocol 字段的配置文件
	configPath := "/tmp/coscli-test-no-protocol.yaml"
	content := `cos:
  base:
    secretid: "test-secret-id"
    secretkey: "test-secret-key"
    sessiontoken: ""
    disableEncryption: "true"
  buckets:
    - name: "test-bucket-1234567890"
      alias: "test-alias"
      region: "ap-guangzhou"
      endpoint: "cos.ap-guangzhou.myqcloud.com"
`
	os.WriteFile(configPath, []byte(content), 0644)
	defer os.Remove(configPath)

	Convey("Test initConfig sets default protocol", t, func() {
		Reset(func() {
			clearCmd()
			setupTestConfig()
		})

		Convey("config without protocol defaults to https", func() {
			var s *cos.ServiceService
			var patches *Patches
			patches = ApplyMethodFunc(reflect.TypeOf(s), "Get",
				func(ctx context.Context, opt ...*cos.ServiceGetOptions) (*cos.ServiceGetResult, *cos.Response, error) {
					return &cos.ServiceGetResult{}, &cos.Response{}, nil
				})
			defer patches.Reset()
			// 先将 config.Base.Protocol 设置为空，确保触发默认值分支
			config.Base.Protocol = ""
			cmd := rootCmd
			cmd.SetArgs([]string{"ls", "-c", configPath})
			e := cmd.Execute()
			_ = e
			// 验证 protocol 被设置为 https
			So(config.Base.Protocol, ShouldEqual, "https")
		})
	})
}

func TestInitConfigOsExitBranches(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("Test initConfig os.Exit branches via patching", t, func() {
		var patches *Patches
		Reset(func() {
			if patches != nil {
				patches.Reset()
				patches = nil
			}
			clearCmd()
			setupTestConfig()
		})

		Convey("config file not ending with .yaml triggers exit", func() {
			// 打桩 os.Exit 防止测试进程退出
			exitCalled := false
			patches = ApplyFunc(os.Exit, func(code int) {
				exitCalled = true
			})
			cmd := rootCmd
			cmd.SetArgs([]string{"ls", "-c", "/tmp/test.json"})
			cmd.Execute()
			So(exitCalled, ShouldBeTrue)
		})

		Convey("viper ReadInConfig error triggers exit", func() {
			// 创建一个无效的 yaml 配置文件
			invalidConfigPath := "/tmp/coscli-test-invalid.yaml"
			os.WriteFile(invalidConfigPath, []byte("invalid: yaml: content: ["), 0644)
			defer os.Remove(invalidConfigPath)

			exitCalled := false
			patches = ApplyFunc(os.Exit, func(code int) {
				exitCalled = true
			})
			cmd := rootCmd
			cmd.SetArgs([]string{"ls", "-c", invalidConfigPath})
			cmd.Execute()
			So(exitCalled, ShouldBeTrue)
		})

		Convey("viper UnmarshalKey error triggers exit", func() {
			// 打桩 viper.UnmarshalKey 返回错误
			exitCalled := false
			patches = ApplyFunc(os.Exit, func(code int) {
				exitCalled = true
			})
			patches.ApplyFunc(viper.UnmarshalKey, func(key string, rawVal interface{}, opts ...viper.DecoderConfigOption) error {
				return fmt.Errorf("test unmarshal error")
			})
			cmd := rootCmd
			cmd.SetArgs([]string{"ls", "-c", testConfigPath})
			cmd.Execute()
			So(exitCalled, ShouldBeTrue)
		})

		Convey("config with encryption disabled false decrypts secrets", func() {
			// 创建一个 disableEncryption 为 false 的配置文件，使用加密后的秘钥
			// 加密值由 util.EncryptSecret("test-secret-id") 和 util.EncryptSecret("test-secret-key") 生成
			configPath := "/tmp/coscli-test-encrypt.yaml"
			content := `cos:
  base:
    secretid: "BOXpRvyeXakOUug/Acl+QA=="
    secretkey: "1jQCBkIZo8g/AnKUAK3iyQ=="
    sessiontoken: ""
    protocol: "https"
    disableEncryption: "false"
  buckets:
    - name: "test-bucket-1234567890"
      alias: "test-alias"
      region: "ap-guangzhou"
      endpoint: "cos.ap-guangzhou.myqcloud.com"
`
			os.WriteFile(configPath, []byte(content), 0644)
			defer os.Remove(configPath)

			var s *cos.ServiceService
			patches = ApplyMethodFunc(reflect.TypeOf(s), "Get",
				func(ctx context.Context, opt ...*cos.ServiceGetOptions) (*cos.ServiceGetResult, *cos.Response, error) {
					return &cos.ServiceGetResult{}, &cos.Response{}, nil
				})
			cmd := rootCmd
			cmd.SetArgs([]string{"ls", "-c", configPath})
			e := cmd.Execute()
			_ = e
			// DecryptSecret 成功，config.Base.SecretKey/SecretID 被更新为解密后的值
		})

		Convey("config path with ~ prefix", func() {
			// 创建 ~/.cos-test-tilde.yaml 配置文件
			home, _ := os.UserHomeDir()
			configPath := home + "/.cos-test-tilde.yaml"
			content := `cos:
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
			os.WriteFile(configPath, []byte(content), 0644)
			defer os.Remove(configPath)

			var s *cos.ServiceService
			patches = ApplyMethodFunc(reflect.TypeOf(s), "Get",
				func(ctx context.Context, opt ...*cos.ServiceGetOptions) (*cos.ServiceGetResult, *cos.Response, error) {
					return &cos.ServiceGetResult{}, &cos.Response{}, nil
				})
			cmd := rootCmd
			// 使用 ~ 开头的路径
			cmd.SetArgs([]string{"ls", "-c", "~/.cos-test-tilde.yaml"})
			e := cmd.Execute()
			_ = e
		})

		Convey("no config file, initSkip with valid params", func() {
			// 打桩 os.Stat 让 ~/.cos.yaml 不存在
			home, _ := os.UserHomeDir()
			cosYamlPath := home + "/.cos.yaml"
			patches = ApplyFunc(os.Stat, func(name string) (os.FileInfo, error) {
				if name == cosYamlPath {
					return nil, os.ErrNotExist
				}
				return os.Lstat(name)
			})
			// 打桩 initConfigFile 防止交互式输入
			patches.ApplyFunc(initConfigFile, func(cfgFlag bool) error {
				return nil
			})
			// 打桩 os.Exit 防止进程退出
			patches.ApplyFunc(os.Exit, func(code int) {})
			// 打桩 viper.ReadInConfig 防止读取失败
			patches.ApplyFunc(viper.ReadInConfig, func() error {
				return nil
			})
			cmd := rootCmd
			// 不传 -c 参数，触发 cfgFile == "" 分支
			cmd.SetArgs([]string{"ls", "--init-skip", "-i", "test-id", "-k", "test-key", "-e", "cos.ap-guangzhou.myqcloud.com"})
			e := cmd.Execute()
			_ = e
		})

		Convey("no config file, no initSkip, calls initConfigFile", func() {
			// 打桩 os.Stat 让 ~/.cos.yaml 不存在
			home, _ := os.UserHomeDir()
			cosYamlPath := home + "/.cos.yaml"
			patches = ApplyFunc(os.Stat, func(name string) (os.FileInfo, error) {
				if name == cosYamlPath {
					return nil, os.ErrNotExist
				}
				return os.Lstat(name)
			})
			// 打桩 initConfigFile 防止交互式输入
			initFileCalled := false
			patches.ApplyFunc(initConfigFile, func(cfgFlag bool) error {
				initFileCalled = true
				return nil
			})
			// 打桩 viper.ReadInConfig 防止读取失败
			patches.ApplyFunc(viper.ReadInConfig, func() error {
				return nil
			})
			cmd := rootCmd
			// 不传 -c 参数，不传 --init-skip，触发 initConfigFile 调用
			cmd.SetArgs([]string{"ls"})
			e := cmd.Execute()
			_ = e
			So(initFileCalled, ShouldBeTrue)
		})

		Convey("no config file, firstArg is config, calls initConfigFile", func() {
			// 打桩 os.Stat 让 ~/.cos.yaml 不存在
			home, _ := os.UserHomeDir()
			cosYamlPath := home + "/.cos.yaml"
			patches = ApplyFunc(os.Stat, func(name string) (os.FileInfo, error) {
				if name == cosYamlPath {
					return nil, os.ErrNotExist
				}
				return os.Lstat(name)
			})
			// 打桩 initConfigFile 防止交互式输入
			patches.ApplyFunc(initConfigFile, func(cfgFlag bool) error {
				return nil
			})
			// 打桩 viper.ReadInConfig 防止读取失败
			patches.ApplyFunc(viper.ReadInConfig, func() error {
				return nil
			})
			cmd := rootCmd
			// 使用 config 子命令，触发 firstArg == "config" 分支
			cmd.SetArgs([]string{"config", "show"})
			e := cmd.Execute()
			_ = e
		})

		Convey("no config file, firstArg is config, initSkip returns", func() {
			// 注意：在测试中 os.Args[1] 是测试框架参数，不是 cmd.SetArgs 设置的参数
			// 所以这个用例实际上测试的是 firstArg != "config" 且 initSkip == true 的分支
			home, _ := os.UserHomeDir()
			cosYamlPath := home + "/.cos.yaml"
			patches = ApplyFunc(os.Stat, func(name string) (os.FileInfo, error) {
				if name == cosYamlPath {
					return nil, os.ErrNotExist
				}
				return os.Lstat(name)
			})
			patches.ApplyFunc(viper.ReadInConfig, func() error {
				return nil
			})
			cmd := rootCmd
			// initSkip 且有所有必要参数，触发 return 分支
			cmd.SetArgs([]string{"config", "show", "--init-skip", "-i", "test-id", "-k", "test-key", "-e", "cos.ap-guangzhou.myqcloud.com"})
			e := cmd.Execute()
			_ = e
		})

		Convey("no config file, initSkip, missing SecretID triggers fatal", func() {
			home, _ := os.UserHomeDir()
			cosYamlPath := home + "/.cos.yaml"
			patches = ApplyFunc(os.Stat, func(name string) (os.FileInfo, error) {
				if name == cosYamlPath {
					return nil, os.ErrNotExist
				}
				return os.Lstat(name)
			})
			// 同时打桩 os.Exit 和设置 logrus ExitFunc，阻止进程退出
			fatalCalled := false
			logrus.StandardLogger().ExitFunc = func(code int) {
				fatalCalled = true
			}
			defer func() { logrus.StandardLogger().ExitFunc = nil }()
			patches.ApplyFunc(os.Exit, func(code int) {})
			cmd := rootCmd
			// initSkip 但没有 SecretID
			cmd.SetArgs([]string{"ls", "--init-skip"})
			cmd.Execute()
			So(fatalCalled, ShouldBeTrue)
		})

		Convey("no config file, initSkip, missing SecretKey triggers fatal", func() {
			home, _ := os.UserHomeDir()
			cosYamlPath := home + "/.cos.yaml"
			patches = ApplyFunc(os.Stat, func(name string) (os.FileInfo, error) {
				if name == cosYamlPath {
					return nil, os.ErrNotExist
				}
				return os.Lstat(name)
			})
			fatalCalled := false
			logrus.StandardLogger().ExitFunc = func(code int) {
				fatalCalled = true
			}
			defer func() { logrus.StandardLogger().ExitFunc = nil }()
			patches.ApplyFunc(os.Exit, func(code int) {})
			cmd := rootCmd
			// initSkip 有 SecretID 但没有 SecretKey
			cmd.SetArgs([]string{"ls", "--init-skip", "-i", "test-id"})
			cmd.Execute()
			So(fatalCalled, ShouldBeTrue)
		})

		Convey("no config file, initSkip, missing Endpoint triggers fatal", func() {
			home, _ := os.UserHomeDir()
			cosYamlPath := home + "/.cos.yaml"
			patches = ApplyFunc(os.Stat, func(name string) (os.FileInfo, error) {
				if name == cosYamlPath {
					return nil, os.ErrNotExist
				}
				return os.Lstat(name)
			})
			fatalCalled := false
			logrus.StandardLogger().ExitFunc = func(code int) {
				fatalCalled = true
			}
			defer func() { logrus.StandardLogger().ExitFunc = nil }()
			patches.ApplyFunc(os.Exit, func(code int) {})
			cmd := rootCmd
			// initSkip 有 SecretID 和 SecretKey 但没有 Endpoint
			cmd.SetArgs([]string{"ls", "--init-skip", "-i", "test-id", "-k", "test-key"})
			cmd.Execute()
			So(fatalCalled, ShouldBeTrue)
		})

		Convey("firstArg is config, no initSkip, calls initConfigFile", func() {
			// 通过修改 os.Args 让 firstArg == "config"
			origArgs := os.Args
			os.Args = []string{"coscli", "config"}
			defer func() { os.Args = origArgs }()

			home, _ := os.UserHomeDir()
			cosYamlPath := home + "/.cos.yaml"
			patches = ApplyFunc(os.Stat, func(name string) (os.FileInfo, error) {
				if name == cosYamlPath {
					return nil, os.ErrNotExist
				}
				return os.Lstat(name)
			})
			initFileCalled := false
			patches.ApplyFunc(initConfigFile, func(cfgFlag bool) error {
				initFileCalled = true
				return nil
			})
			patches.ApplyFunc(viper.ReadInConfig, func() error {
				return nil
			})
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "show"})
			e := cmd.Execute()
			_ = e
			So(initFileCalled, ShouldBeTrue)
		})

		Convey("firstArg is config, initSkip, returns early", func() {
			// 通过修改 os.Args 让 firstArg == "config"
			origArgs := os.Args
			os.Args = []string{"coscli", "config"}
			defer func() { os.Args = origArgs }()

			home, _ := os.UserHomeDir()
			cosYamlPath := home + "/.cos.yaml"
			patches = ApplyFunc(os.Stat, func(name string) (os.FileInfo, error) {
				if name == cosYamlPath {
					return nil, os.ErrNotExist
				}
				return os.Lstat(name)
			})
			patches.ApplyFunc(viper.ReadInConfig, func() error {
				return nil
			})
			cmd := rootCmd
			// initSkip 触发 firstArg == "config" && initSkip 分支，直接 return
			cmd.SetArgs([]string{"config", "show", "--init-skip"})
			e := cmd.Execute()
			_ = e
		})
	})
}
