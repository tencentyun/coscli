package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/mitchellh/go-homedir"
	logger "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var testDir = "test-tmp-dir"

var appID string

var testBucket string
var testAlias string
var testBucket1 string
var testAlias1 string
var testBucket2 string
var testAlias2 string
var testEndpoint = "cos.ap-guangzhou.myqcloud.com"

var testOfsBucket string
var testOfsBucketAlias string

func init() {
	// 读取配置文件
	getConfig()
	// 初始化 app-id
	name := config.Buckets[0].Name
	if len(name) > 10 {
		appID = name[len(name)-10:]
	}
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

func randStr(length int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyz"
	bytes := []byte(str)
	result := []byte{}
	rand.Seed(time.Now().UnixNano() + int64(rand.Intn(100)))
	for i := 0; i < length; i++ {
		result = append(result, bytes[rand.Intn(len(bytes))])
	}
	return string(result)
}

func setUp(testBucket, testAlias, testEndpoint string, ofs bool) {
	// 创建测试桶
	logger.Infoln(fmt.Sprintf("创建测试桶：%s-%s %s", testBucket, appID, testEndpoint))
	clearCmd()
	cmd := rootCmd
	var args []string
	if ofs {
		args = []string{"mb",
			fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint, "-o"}
	} else {
		args = []string{"mb",
			fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint}
	}
	cmd.SetArgs(args)
	err := cmd.Execute()
	if err != nil {
		logger.Errorln(err)
	}

	if testAlias == "nil" {
		return
	}

	// 更新配置文件
	logger.Infoln(fmt.Sprintf("更新配置文件：%s", testBucket))
	if testAlias == "" {
		args = []string{"config", "add", "-b",
			fmt.Sprintf("%s-%s", testBucket, appID), "-e", testEndpoint}
	} else {
		args = []string{"config", "add", "-b",
			fmt.Sprintf("%s-%s", testBucket, appID), "-e", testEndpoint, "-a", testAlias}
	}
	if ofs {
		args = append(args, "-o")
	}
	clearCmd()
	cmd = rootCmd
	cmd.SetArgs(args)
	err = cmd.Execute()
	if err != nil {
		logger.Errorln(err)
	}

	// 更新 Config
	getConfig()
}

func tearDown(testBucket, testAlias, testEndpoint string) {
	if testAlias == "" {
		testAlias = testBucket + "-" + appID
	}
	// 清空测试桶
	logger.Infoln(fmt.Sprintf("清空测试桶文件：%s", testAlias))
	args := []string{"rm",
		fmt.Sprintf("cos://%s", testAlias), "-r", "-f"}
	clearCmd()
	cmd := rootCmd
	cmd.SetArgs(args)
	err := cmd.Execute()
	if err != nil {
		logger.Errorln(err)
	}
	logger.Infoln(fmt.Sprintf("清空测试桶碎片：%s", testAlias))
	args = []string{"abort",
		fmt.Sprintf("cos://%s", testAlias)}
	clearCmd()
	cmd = rootCmd
	cmd.SetArgs(args)
	err = cmd.Execute()
	if err != nil {
		logger.Errorln(err)
	}

	// 删除测试桶
	logger.Infoln(fmt.Sprintf("删除测试桶：%s-%s %s", testBucket, appID, testEndpoint))
	args = []string{"rb",
		fmt.Sprintf("cos://%s-%s", testBucket, appID), "-e", testEndpoint}
	clearCmd()
	cmd = rootCmd
	cmd.SetArgs(args)
	err = cmd.Execute()
	if err != nil {
		logger.Errorln(err)
	}

	// 更新配置文件
	logger.Infoln(fmt.Sprintf("更新配置文件：%s", testAlias))
	args = []string{"config", "delete", "-a", testAlias}
	clearCmd()
	cmd = rootCmd
	cmd.SetArgs(args)
	err = cmd.Execute()
	if err != nil {
		logger.Errorln(err)
	}
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

func clearCmd() {
	rootCmd.Flags().VisitAll(func(flag *pflag.Flag) {
		flag.Value.Set(flag.DefValue)
	})

	// 重置子命令的状态
	for _, subCmd := range rootCmd.Commands() {
		subCmd.Flags().VisitAll(func(flag *pflag.Flag) {
			flag.Value.Set(flag.DefValue)
		})
	}
}
