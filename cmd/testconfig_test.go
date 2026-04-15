package cmd

import (
	"os"

	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var testEndpoint = "cos.ap-guangzhou.myqcloud.com"

const testConfigPath = "/tmp/coscli-test.yaml"

// setupTestConfig 创建临时测试配置文件，并加载到全局 config
func setupTestConfig() {
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
      ofs: false
      customized: false
    - name: "test-bucket2-1234567890"
      alias: "test-alias2"
      region: "ap-guangzhou"
      endpoint: "cos.ap-guangzhou.myqcloud.com"
      ofs: false
      customized: false
`
	if err := os.WriteFile(testConfigPath, []byte(content), 0644); err != nil {
		logger.Errorln("创建测试配置文件失败:", err)
		return
	}
	// 重置 viper 状态，避免之前的 viper.Set 覆盖文件中的值
	viper.Reset()
	viper.SetConfigFile(testConfigPath)
	if err := viper.ReadInConfig(); err != nil {
		logger.Errorln("读取测试配置文件失败:", err)
		return
	}
	if err := viper.UnmarshalKey("cos", &config); err != nil {
		logger.Errorln("解析测试配置文件失败:", err)
	}
}

// teardownTestConfig 删除临时测试配置文件
func teardownTestConfig() {
	if err := os.Remove(testConfigPath); err != nil && !os.IsNotExist(err) {
		logger.Errorln("删除测试配置文件失败:", err)
	}
}

func clearCmd() {
	var resetFlags func(cmd *cobra.Command)
	resetFlags = func(cmd *cobra.Command) {
		cmd.Flags().VisitAll(func(flag *pflag.Flag) {
			flag.Value.Set(flag.DefValue)
		})
		for _, subCmd := range cmd.Commands() {
			resetFlags(subCmd)
		}
	}
	resetFlags(rootCmd)
}
