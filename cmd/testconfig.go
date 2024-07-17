package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/mitchellh/go-homedir"
	logger "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var testDir = "test-tmp-dir"

var appID string

var testBucket = "coscli-test"
var testAlias = "coscli-test-alias"
var testEndpoint = "cos.ap-guangzhou.myqcloud.com"

var testBucket1 = "coscli-test1"
var testAlias1 = "coscli-test1-alias"
var testEndpoint1 = "cos.ap-guangzhou.myqcloud.com"

var testBucket2 = "coscli-test2"
var testAlias2 = "coscli-test2-alias"
var testEndpoint2 = "cos.ap-guangzhou.myqcloud.com"

var testOfsBucket = "ofstest"

func init() {
	// 读取配置文件
	getConfig()
	// 初始化 app-id
	name := config.Buckets[0].Name
	appID = name[len(name)-10:]
}

func getConfig() {
	home, err := homedir.Dir()
	if err != nil {
		logger.Errorln(err)
	}
	viper.SetConfigFile(home + "/.cos.yaml")

	if err = viper.ReadInConfig(); err != nil {
		logger.Errorln(err)
	}
	if err = viper.UnmarshalKey("cos", &config); err != nil {
		logger.Errorln(err)
	}
}

func setUp(testBucket, testAlias, testEndpoint string) {
	// 创建测试桶
	logger.Infoln(fmt.Sprintf("创建测试桶：%s-%s %s", testBucket, appID, testEndpoint))
	cmd := rootCmd
	args := []string{"mb",
		fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint}
	cmd.SetArgs(args)
	err := cmd.Execute()
	if err != nil {
		logger.Errorln("SetUp error: 创建测试桶失败")
	}

	// 更新配置文件
	logger.Infoln(fmt.Sprintf("更新配置文件：%s", testAlias))
	args = []string{"config", "add", "-b",
		fmt.Sprintf("%s-%s", testBucket, appID), "-e", testEndpoint, "-a", testAlias}
	cmd.SetArgs(args)
	err = cmd.Execute()
	if err != nil {
		logger.Errorln("SetUp error: 更新配置文件失败")
	}

	// 更新 Config
	getConfig()
}

func tearDown(testBucket, testAlias, testEndpoint string) {
	// 清空测试桶
	logger.Infoln(fmt.Sprintf("清空测试桶：%s", testAlias))
	cmd := rootCmd
	args := []string{"rm",
		fmt.Sprintf("cos://%s", testAlias), "-r", "-f"}
	cmd.SetArgs(args)
	err := cmd.Execute()
	if err != nil {
		logger.Errorln("TearDown error: 清空测试桶失败")
	}
	args = []string{"abort",
		fmt.Sprintf("cos://%s", testAlias)}
	cmd.SetArgs(args)
	err = cmd.Execute()
	if err != nil {
		logger.Errorln("TearDown error: 清空测试桶失败")
	}

	// 删除测试桶
	logger.Infoln(fmt.Sprintf("删除测试桶：%s-%s %s", testBucket, appID, testEndpoint))
	args = []string{"rb",
		fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint}
	cmd.SetArgs(args)
	err = cmd.Execute()
	if err != nil {
		logger.Errorln("TearDown error: 删除测试桶失败")
	}

	// 更新配置文件
	logger.Infoln(fmt.Sprintf("更新配置文件：%s", testAlias))
	args = []string{"config", "delete", "-a",
		fmt.Sprintf("%s", testAlias)}
	cmd.SetArgs(args)
	err = cmd.Execute()
	if err != nil {
		logger.Errorln("TearDown error: 更新配置文件失败")
	}
}

func createTestBucket(testBucket, testEndpoint string) {
	logger.Infoln(fmt.Sprintf("创建测试桶：%s-%s %s", testBucket, appID, testEndpoint))
	cmd := rootCmd
	args := []string{"mb",
		fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint}
	cmd.SetArgs(args)
	err := cmd.Execute()
	if err != nil {
		logger.Errorln("SetUp error: 创建测试桶失败")
	}
}

func deleteTestBucket(testBucket, testEndpoint string) {
	logger.Infoln(fmt.Sprintf("删除测试桶：%s-%s %s", testBucket, appID, testEndpoint))
	cmd := rootCmd
	args := []string{"rb",
		fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint}
	cmd.SetArgs(args)
	err := cmd.Execute()
	if err != nil {
		logger.Errorln("TearDown error: 删除测试桶失败")
	}
}

func addConfig(testBucket, testEndpoint string) {
	cmd := rootCmd
	args := []string{"config", "add", "-b",
		fmt.Sprintf("%s-%s", testBucket, appID), "-e", testEndpoint}
	cmd.SetArgs(args)
	err := cmd.Execute()
	if err != nil {
		logger.Errorln("SetUp error: 更新配置文件失败")
	}

	// 更新 Config
	getConfig()
}

func addConfig_alias(testBucket, testAlias, testEndpoint string) {
	cmd := rootCmd
	args := []string{"config", "add", "-b",
		fmt.Sprintf("%s-%s", testBucket, appID), "-e", testEndpoint, "-a", testAlias}
	cmd.SetArgs(args)
	err := cmd.Execute()
	if err != nil {
		logger.Errorln("SetUp error: 更新配置文件失败")
	}

	// 更新 Config
	getConfig()
}

func deleteConfig(testBucket string) {
	cmd := rootCmd
	args := []string{"config", "delete", "-a",
		fmt.Sprintf("%s-%s", testBucket, appID)}
	cmd.SetArgs(args)
	err := cmd.Execute()
	if err != nil {
		logger.Errorln("TearDown error: 更新配置文件失败")
	}

	// 更新 Config
	getConfig()
}

func deleteConfig_alias(testAlias string) {
	cmd := rootCmd
	args := []string{"config", "delete", "-a", testAlias}
	cmd.SetArgs(args)
	err := cmd.Execute()
	if err != nil {
		logger.Errorln("TearDown error: 更新配置文件失败")
	}

	// 更新 Config
	getConfig()
}

// 创建文件
func genFile(fileName string, size int) {
	data := make([]byte, 0)

	rand.Seed(time.Now().Unix())
	for i := 0; i < size; i++ {
		u := uint8(rand.Intn(256))
		data = append(data, u)
	}

	f, err := os.Create(fileName)
	if err != nil {
		logger.Errorln("genFile error: 创建文件失败")
	}
	defer f.Close()

	n, err := f.Write(data)
	if err != nil || n != size {
		logger.Errorln("genFile error: 数据写入失败")
	}
}

// 创建目录，有 num 个小文件和3个大文件
func genDir(dirName string, num int) {
	if err := os.MkdirAll(fmt.Sprintf("%s/small-file", dirName), os.ModePerm); err != nil {
		logger.Errorln("genDir error: 创建文件夹失败")
	}
	if err := os.MkdirAll(fmt.Sprintf("%s/big-file", dirName), os.ModePerm); err != nil {
		logger.Errorln("genDir error: 创建文件夹失败")
	}

	logger.Infoln(fmt.Sprintf("生成小文件：%s/small-file", dirName))
	for i := 0; i < num; i++ {
		genFile(fmt.Sprintf("%s/small-file/%d", dirName, i), 30*1024)
	}
	logger.Infoln(fmt.Sprintf("生成大文件：%s/big-file", dirName))
	for i := 0; i < 3; i++ {
		genFile(fmt.Sprintf("%s/big-file/%d", dirName, i), 5*1024*1024)
	}
}

func delDir(dirName string) {
	logger.Infoln(fmt.Sprintf("删除测试临时文件夹：%s", dirName))
	if err := os.RemoveAll(dirName); err != nil {
		logger.Errorln("delDir error: 删除文件夹失败")
	}
}
