package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	"github.com/mitchellh/go-homedir"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/viper"
)

func TestConfigInitCmd(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("Test coscli config init", t, func() {
		var patches *Patches
		Reset(func() {
			if patches != nil {
				patches.Reset()
				patches = nil
			}
			clearCmd()
		})

		Convey("initConfigFile error", func() {
			patches = ApplyFunc(initConfigFile, func(cfgFlag bool) error {
				return fmt.Errorf("test initConfigFile error")
			})
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "init", "-c", testConfigPath})
			e := cmd.Execute()
			fmt.Printf(" : %v", e)
			So(e, ShouldBeError)
		})
	})
}

func TestInitConfigFile(t *testing.T) {
	originalStdin := os.Stdin
	defer func() {
		os.Stdin = originalStdin
	}()

	originalViper := *viper.GetViper()
	defer func() {
		*viper.GetViper() = originalViper
	}()

	originalCmdCnt := cmdCnt
	defer func() {
		cmdCnt = originalCmdCnt
	}()

	viper.Reset()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "testinit.yaml")

	inputs := []string{
		configPath,
		"SecretKey",
		"test-secret-id",
		"test-secret-key",
		"",
		"true",
		"false",
		"false",
		"test-bucket-1234567890",
		"cos.ap-beijing.myqcloud.com",
		"test-alias",
		"false",
	}
	inputBuffer := bytes.NewBufferString("")
	for _, input := range inputs {
		inputBuffer.WriteString(input + "\n")
	}

	r, w, _ := os.Pipe()
	os.Stdin = r
	go func() {
		defer w.Close()
		io.Copy(w, inputBuffer)
	}()

	err := initConfigFile(true)

	Convey("TestInitConfigFile", t, func() {
		So(err, ShouldBeNil)
		v := viper.New()
		v.SetConfigFile(configPath)
		readErr := v.ReadInConfig()
		So(readErr, ShouldBeNil)
		So(v.GetString("cos.base.mode"), ShouldEqual, "SecretKey")
	})
}

func TestInitConfigFileCvmRole(t *testing.T) {
	originalStdin := os.Stdin
	defer func() {
		os.Stdin = originalStdin
	}()

	viper.Reset()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "testinit-cvmrole.yaml")

	inputs := []string{
		configPath,
		"CvmRole",
		"my-cvm-role",
		"true",
		"false",
		"false",
		"test-bucket-1234567890",
		"cos.ap-beijing.myqcloud.com",
		"test-alias",
		"false",
	}
	inputBuffer := bytes.NewBufferString("")
	for _, input := range inputs {
		inputBuffer.WriteString(input + "\n")
	}

	r, w, _ := os.Pipe()
	os.Stdin = r
	go func() {
		defer w.Close()
		io.Copy(w, inputBuffer)
	}()

	err := initConfigFile(true)

	Convey("TestInitConfigFileCvmRole", t, func() {
		So(err, ShouldBeNil)
	})
}

func TestInitConfigFileWriteError(t *testing.T) {
	originalStdin := os.Stdin
	defer func() {
		os.Stdin = originalStdin
	}()

	viper.Reset()

	configPath := "/nonexistent-dir/testinit-error.yaml"

	inputs := []string{
		configPath,
		"SecretKey",
		"test-secret-id",
		"test-secret-key",
		"",
		"true",
		"false",
		"false",
		"test-bucket-1234567890",
		"cos.ap-beijing.myqcloud.com",
		"test-alias",
		"false",
	}
	inputBuffer := bytes.NewBufferString("")
	for _, input := range inputs {
		inputBuffer.WriteString(input + "\n")
	}

	r, w, _ := os.Pipe()
	os.Stdin = r
	go func() {
		defer w.Close()
		io.Copy(w, inputBuffer)
	}()

	err := initConfigFile(true)

	Convey("TestInitConfigFileWriteError", t, func() {
		So(err, ShouldBeError)
	})
}

func TestInitConfigFileWithEncryption(t *testing.T) {
	originalStdin := os.Stdin
	defer func() {
		os.Stdin = originalStdin
	}()

	viper.Reset()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "testinit-enc.yaml")

	// disableEncryption 为 "false"，触发加密分支
	inputs := []string{
		configPath,
		"SecretKey",
		"test-secret-id",
		"test-secret-key",
		"",
		"false", // disableEncryption = false，触发加密
		"false",
		"false",
		"test-bucket-1234567890",
		"cos.ap-beijing.myqcloud.com",
		"test-alias",
		"false",
	}
	inputBuffer := bytes.NewBufferString("")
	for _, input := range inputs {
		inputBuffer.WriteString(input + "\n")
	}

	r, w, _ := os.Pipe()
	os.Stdin = r
	go func() {
		defer w.Close()
		io.Copy(w, inputBuffer)
	}()

	err := initConfigFile(true)

	Convey("TestInitConfigFileWithEncryption", t, func() {
		So(err, ShouldBeNil)
	})
}

func TestInitConfigFileTildePath(t *testing.T) {
	originalStdin := os.Stdin
	defer func() {
		os.Stdin = originalStdin
	}()

	viper.Reset()

	// TestMain 已把 HOME 切到受控临时目录，~ 会被解析到该临时目录下，不会污染用户。
	home, _ := homedir.Dir()
	configPath := "~/.cos-test-init.yaml"
	defer os.Remove(home + "/.cos-test-init.yaml")

	inputs := []string{
		configPath,
		"SecretKey",
		"test-secret-id",
		"test-secret-key",
		"",
		"true",
		"false",
		"false",
		"test-bucket-1234567890",
		"cos.ap-beijing.myqcloud.com",
		"test-alias",
		"false",
	}
	inputBuffer := bytes.NewBufferString("")
	for _, input := range inputs {
		inputBuffer.WriteString(input + "\n")
	}

	r, w, _ := os.Pipe()
	os.Stdin = r
	go func() {
		defer w.Close()
		io.Copy(w, inputBuffer)
	}()

	err := initConfigFile(true)

	Convey("TestInitConfigFileTildePath", t, func() {
		So(err, ShouldBeNil)
	})
}

func TestConfigInitCmdCntSkip(t *testing.T) {
	setupTestConfig()
	defer teardownTestConfig()

	Convey("Test config init skipped when cmdCnt >= 1", t, func() {
		Reset(func() {
			clearCmd()
			setupTestConfig()
		})

		Convey("config init skipped when cmdCnt >= 1", func() {
			// 设置 cmdCnt >= 1，触发 return nil 分支
			cmdCnt = 1
			defer func() { cmdCnt = 0 }()
			cmd := rootCmd
			cmd.SetArgs([]string{"config", "init", "-c", testConfigPath})
			e := cmd.Execute()
			So(e, ShouldBeNil)
		})
	})
}

func TestInitConfigFileEmptyAlias(t *testing.T) {
	originalStdin := os.Stdin
	defer func() {
		os.Stdin = originalStdin
	}()

	viper.Reset()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "testinit-empty-alias.yaml")

	// bucket.Alias 为空，触发 bucket.Alias = bucket.Name 分支
	inputs := []string{
		configPath,
		"SecretKey",
		"test-secret-id",
		"test-secret-key",
		"",
		"true",
		"false",
		"false",
		"test-bucket-1234567890",
		"cos.ap-beijing.myqcloud.com",
		"", // 空 alias，触发 bucket.Alias = bucket.Name
		"false",
	}
	inputBuffer := bytes.NewBufferString("")
	for _, input := range inputs {
		inputBuffer.WriteString(input + "\n")
	}

	r, w, _ := os.Pipe()
	os.Stdin = r
	go func() {
		defer w.Close()
		io.Copy(w, inputBuffer)
	}()

	err := initConfigFile(true)

	Convey("TestInitConfigFileEmptyAlias", t, func() {
		So(err, ShouldBeNil)
	})
}

func TestInitConfigFileTildePathPatched(t *testing.T) {
	// 通过打桩 fmt.Scanf 来模拟用户输入 ~ 路径
	// 这样可以覆盖 config_init.go:48 的 configFile[0] == '~' 分支
	// TestMain 已把 HOME 切到受控临时目录，~ 解析后位于该目录下，不污染用户家目录。
	viper.Reset()

	home, _ := homedir.Dir()
	tildeConfigPath := "~/.cos-test-init-patched.yaml"
	realConfigPath := home + "/.cos-test-init-patched.yaml"
	defer os.Remove(realConfigPath)

	callCount := 0
	inputs := []string{
		tildeConfigPath,               // 第1次调用：configFile = "~/.cos-test-init-patched.yaml"
		"SecretKey",                   // 第2次调用：mode
		"test-secret-id",              // 第3次调用：secretID
		"test-secret-key",             // 第4次调用：secretKey
		"",                            // 第5次调用：sessionToken
		"true",                        // 第6次调用：disableEncryption
		"false",                       // 第7次调用：closeAutoSwitchHost
		"false",                       // 第8次调用：disableAutoFetchBucketType
		"test-bucket-1234567890",      // 第9次调用：bucket name
		"cos.ap-beijing.myqcloud.com", // 第10次调用：endpoint
		"test-alias",                  // 第11次调用：alias
		"false",                       // 第12次调用：customized
		"false",                       // 第13次调用：add more buckets
	}

	patches := ApplyFunc(fmt.Scanf, func(format string, a ...interface{}) (int, error) {
		if callCount < len(inputs) {
			input := inputs[callCount]
			callCount++
			if len(a) > 0 {
				if ptr, ok := a[0].(*string); ok {
					*ptr = input
				}
			}
			return 1, nil
		}
		return 0, io.EOF
	})
	defer patches.Reset()

	err := initConfigFile(true)

	Convey("TestInitConfigFileTildePathPatched", t, func() {
		So(err, ShouldBeNil)
	})
}
