package main

import (
	"coscli/util"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"

	logger "github.com/sirupsen/logrus"
	. "github.com/smartystreets/goconvey/convey"
)

var testDir = "test-tmp-dir"
var config util.Config
var param util.Param
var appID string

var testBucket1 = "coscli-test1"
var testAlias1 = "coscli-test1"
var testEndpoint1 = "cos.ap-shanghai.myqcloud.com"

var testBucket2 = "coscli-test2"
var testAlias2 = "coscli-test2"
var testEndpoint2 = "cos.ap-guangzhou.myqcloud.com"

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
		panic(err)
	}
	viper.SetConfigFile(home + "/.cos.yaml")

	if err = viper.ReadInConfig(); err != nil {
		panic(err)
	}
	if err = viper.UnmarshalKey("cos", &config); err != nil {
		panic(err)
	}
}

func setUp(testBucket, testAlias, testEndpoint string) {
	// 创建测试桶
	logger.Infoln(fmt.Sprintf("创建测试桶：%s-%s %s", testBucket, appID, testEndpoint))
	cmd := exec.Command("bash", "-c",
		fmt.Sprintf("./coscli mb cos://%s-%s -e %s", testBucket, appID, testEndpoint))
	if err := cmd.Run(); err != nil {
		panic("SetUp error: 创建测试桶失败")
	}

	// 更新配置文件
	logger.Infoln(fmt.Sprintf("更新配置文件：%s", testAlias))
	cmd = exec.Command("bash", "-c",
		fmt.Sprintf("./coscli config add -b %s-%s -e %s -a %s", testBucket, appID, testEndpoint, testAlias))
	if err := cmd.Run(); err != nil {
		panic("SetUp error: 更新配置文件失败")
	}

	// 更新 Config
	getConfig()
}

func tearDown(testBucket, testAlias, testEndpoint string) {
	// 清空测试桶
	logger.Infoln(fmt.Sprintf("清空测试桶：%s", testAlias))
	cmd := exec.Command("bash", "-c",
		fmt.Sprintf("./coscli rm cos://%s -r -f", testAlias))
	if err := cmd.Run(); err != nil {
		panic("TearDown error: 清空测试桶失败")
	}
	cmd = exec.Command("bash", "-c",
		fmt.Sprintf("./coscli abort cos://%s", testAlias))
	if err := cmd.Run(); err != nil {
		panic("TearDown error: 清空测试桶失败")
	}

	// 删除测试桶
	logger.Infoln(fmt.Sprintf("删除测试桶：%s-%s %s", testBucket, appID, testEndpoint))
	cmd = exec.Command("bash", "-c",
		fmt.Sprintf("./coscli rb cos://%s-%s -e %s", testBucket, appID, testEndpoint))
	if err := cmd.Run(); err != nil {
		panic("TearDown error: 删除测试桶失败")
	}

	// 更新配置文件
	logger.Infoln(fmt.Sprintf("更新配置文件：%s", testAlias))
	cmd = exec.Command("bash", "-c",
		fmt.Sprintf("./coscli config delete -a %s", testAlias))
	if err := cmd.Run(); err != nil {
		panic("TearDown error: 更新配置文件失败")
	}
}

func genFile(fileName string, size int) {
	data := make([]byte, 0)

	rand.Seed(time.Now().Unix())
	for i := 0; i < size; i++ {
		u := uint8(rand.Intn(256))
		data = append(data, u)
	}

	f, err := os.Create(fileName)
	if err != nil {
		panic("genFile error: 创建文件失败")
	}
	defer f.Close()

	n, err := f.Write(data)
	if err != nil || n != size {
		panic("genFile error: 数据写入失败")
	}
}

func genDir(dirName string, num int) {
	if err := os.MkdirAll(fmt.Sprintf("%s/small-file", dirName), os.ModePerm); err != nil {
		panic("genDir error: 创建文件夹失败")
	}
	if err := os.MkdirAll(fmt.Sprintf("%s/big-file", dirName), os.ModePerm); err != nil {
		panic("genDir error: 创建文件夹失败")
	}

	logger.Infoln(fmt.Sprintf("生成小文件：%s/small-file", dirName))
	for i := 0; i < num; i++ {
		genFile(fmt.Sprintf("%s/small-file/%d", dirName, i), 30*1024)
	}
	logger.Infoln(fmt.Sprintf("生成大文件：%s/big-file", dirName))
	for i := 0; i < 3; i++ {
		genFile(fmt.Sprintf("%s/big-file/%d", dirName, i), 40*1024*1024)
	}
}

func delDir(dirName string) {
	logger.Infoln(fmt.Sprintf("删除测试临时文件夹：%s", dirName))
	if err := os.RemoveAll(dirName); err != nil {
		panic("delDir error: 删除文件夹失败")
	}
}

func getCRC(cosPath string) string {
	bucketName, key := util.ParsePath(cosPath)
	param.Endpoint = "cos.ap-guangzhou.myqcloud.com"
	c := util.NewClient(&config, &param, bucketName)
	h, _ := util.ShowHash(c, key, "crc64")
	return h
}

func TestCpCmd(t *testing.T) {
	setUp(testBucket1, testAlias1, testEndpoint1)
	defer tearDown(testBucket1, testAlias1, testEndpoint1)
	setUp(testBucket2, testAlias2, testEndpoint2)
	defer tearDown(testBucket2, testAlias2, testEndpoint2)
	genDir(testDir, 10)
	defer delDir(testDir)

	Convey("Test coscli cp", t, func() {
		Convey("Upload", func() {
			Convey("上传单个小文件", func() {
				localFileName := fmt.Sprintf("%s/small-file/0", testDir)
				cosFileName := fmt.Sprintf("cos://%s/%s", testAlias1, "single-small")

				cmd := exec.Command("bash", "-c",
					fmt.Sprintf("./coscli cp %s %s", localFileName, cosFileName))
				e := cmd.Run()
				So(e, ShouldBeNil)
			})
			Convey("上传多个小文件", func() {
				localFileName := fmt.Sprintf("%s/small-file", testDir)
				cosFileName := fmt.Sprintf("cos://%s/%s", testAlias1, "multi-small")

				cmd := exec.Command("bash", "-c",
					fmt.Sprintf("./coscli cp %s %s -r", localFileName, cosFileName))
				e := cmd.Run()
				So(e, ShouldBeNil)
			})
			Convey("上传单个大文件", func() {
				localFileName := fmt.Sprintf("%s/big-file/0", testDir)
				cosFileName := fmt.Sprintf("cos://%s/%s", testAlias1, "single-big")

				cmd := exec.Command("bash", "-c",
					fmt.Sprintf("./coscli cp %s %s", localFileName, cosFileName))
				e := cmd.Run()
				So(e, ShouldBeNil)
			})
			Convey("上传多个大文件", func() {
				localFileName := fmt.Sprintf("%s/big-file", testDir)
				cosFileName := fmt.Sprintf("cos://%s/%s", testAlias1, "multi-big")

				cmd := exec.Command("bash", "-c",
					fmt.Sprintf("./coscli cp %s %s -r", localFileName, cosFileName))
				e := cmd.Run()
				So(e, ShouldBeNil)
			})
		})

		Convey("Copy", func() {
			Convey("桶内拷贝单个文件", func() {
				srcPath := fmt.Sprintf("cos://%s/%s", testAlias1, "single-big")
				dstPath := fmt.Sprintf("cos://%s/%s", testAlias1, "single-copy")

				cmd := exec.Command("bash", "-c",
					fmt.Sprintf("./coscli cp %s %s", srcPath, dstPath))
				e := cmd.Run()
				So(e, ShouldBeNil)
			})
			Convey("桶内拷贝多个文件", func() {
				srcPath := fmt.Sprintf("cos://%s/%s", testAlias1, "multi-big")
				dstPath := fmt.Sprintf("cos://%s/%s", testAlias1, "multi-copy")

				cmd := exec.Command("bash", "-c",
					fmt.Sprintf("./coscli cp %s %s -r", srcPath, dstPath))
				e := cmd.Run()
				So(e, ShouldBeNil)
			})
			Convey("跨桶拷贝单个小文件", func() {
				srcPath := fmt.Sprintf("cos://%s/%s", testAlias1, "single-small")
				dstPath := fmt.Sprintf("cos://%s/%s", testAlias2, "single-copy-small")

				cmd := exec.Command("bash", "-c",
					fmt.Sprintf("./coscli cp %s %s", srcPath, dstPath))
				e := cmd.Run()
				So(e, ShouldBeNil)
			})
			Convey("跨桶拷贝多个小文件", func() {
				srcPath := fmt.Sprintf("cos://%s/%s", testAlias1, "multi-small")
				dstPath := fmt.Sprintf("cos://%s/%s", testAlias2, "multi-copy-small")

				cmd := exec.Command("bash", "-c",
					fmt.Sprintf("./coscli cp %s %s -r", srcPath, dstPath))
				e := cmd.Run()
				So(e, ShouldBeNil)
			})
			Convey("跨桶拷贝单个大文件", func() {
				srcPath := fmt.Sprintf("cos://%s/%s", testAlias1, "single-big")
				dstPath := fmt.Sprintf("cos://%s/%s", testAlias2, "single-copy-big")

				cmd := exec.Command("bash", "-c",
					fmt.Sprintf("./coscli cp %s %s", srcPath, dstPath))
				e := cmd.Run()
				So(e, ShouldBeNil)
			})
			Convey("跨桶拷贝多个大文件", func() {
				srcPath := fmt.Sprintf("cos://%s/%s", testAlias1, "multi-big")
				dstPath := fmt.Sprintf("cos://%s/%s", testAlias2, "multi-copy-big")

				cmd := exec.Command("bash", "-c",
					fmt.Sprintf("./coscli cp %s %s -r", srcPath, dstPath))
				e := cmd.Run()
				So(e, ShouldBeNil)
			})
		})

		Convey("Download", func() {
			Convey("下载单个小文件", func() {
				localFileName := fmt.Sprintf("%s/download/single-small", testDir)
				cosFileName := fmt.Sprintf("cos://%s/%s", testAlias2, "single-copy-small")

				cmd := exec.Command("bash", "-c",
					fmt.Sprintf("./coscli cp %s %s", cosFileName, localFileName))
				e := cmd.Run()
				So(e, ShouldBeNil)
			})
			Convey("下载多个小文件", func() {
				localFileName := fmt.Sprintf("%s/download/small-file", testDir)
				cosFileName := fmt.Sprintf("cos://%s/%s", testAlias2, "multi-copy-small")

				cmd := exec.Command("bash", "-c",
					fmt.Sprintf("./coscli cp %s %s -r", cosFileName, localFileName))
				e := cmd.Run()
				So(e, ShouldBeNil)
			})
			Convey("下载单个大文件", func() {
				localFileName := fmt.Sprintf("%s/download/single-big", testDir)
				cosFileName := fmt.Sprintf("cos://%s/%s", testAlias2, "single-copy-big")

				cmd := exec.Command("bash", "-c",
					fmt.Sprintf("./coscli cp %s %s", cosFileName, localFileName))
				e := cmd.Run()
				So(e, ShouldBeNil)
			})
			Convey("下载多个大文件", func() {
				localFileName := fmt.Sprintf("%s/download/big-file", testDir)
				cosFileName := fmt.Sprintf("cos://%s/%s", testAlias2, "multi-copy-big")

				cmd := exec.Command("bash", "-c",
					fmt.Sprintf("./coscli cp %s %s -r", cosFileName, localFileName))
				e := cmd.Run()
				So(e, ShouldBeNil)
			})
		})

		Convey("Hash", func() {
			Convey("单个小文件哈希校验", func() {
				hash1, _ := util.CalculateHash(fmt.Sprintf("%s/small-file/0", testDir), "crc64")
				hash2, _ := util.CalculateHash(fmt.Sprintf("%s/download/single-small", testDir), "crc64")
				So(hash1, ShouldEqual, hash2)
			})
			Convey("单个大文件哈希校验", func() {
				hash1, _ := util.CalculateHash(fmt.Sprintf("%s/big-file/0", testDir), "crc64")
				hash2, _ := util.CalculateHash(fmt.Sprintf("%s/download/single-big", testDir), "crc64")
				So(hash1, ShouldEqual, hash2)
			})
			Convey("多个小文件哈希校验", func() {
				fileList1 := util.GetLocalFilesListRecursive(fmt.Sprintf("%s/small-file", testDir), "", "")
				fileList2 := util.GetLocalFilesListRecursive(fmt.Sprintf("%s/download/small-file", testDir), "", "")
				So(len(fileList1), ShouldEqual, len(fileList2))
				for i := 0; i < len(fileList1); i++ {
					hash1, _ := util.CalculateHash(fmt.Sprintf("%s/small-file/%s", testDir, fileList1[i]), "crc64")
					hash2, _ := util.CalculateHash(fmt.Sprintf("%s/download/small-file/%s", testDir, fileList2[i]), "crc64")
					So(hash1, ShouldEqual, hash2)
				}
			})
			Convey("多个大文件哈希校验", func() {
				fileList1 := util.GetLocalFilesListRecursive(fmt.Sprintf("%s/big-file", testDir), "", "")
				fileList2 := util.GetLocalFilesListRecursive(fmt.Sprintf("%s/download/big-file", testDir), "", "")
				So(len(fileList1), ShouldEqual, len(fileList2))
				for i := 0; i < len(fileList1); i++ {
					hash1, _ := util.CalculateHash(fmt.Sprintf("%s/big-file/%s", testDir, fileList1[i]), "crc64")
					hash2, _ := util.CalculateHash(fmt.Sprintf("%s/download/big-file/%s", testDir, fileList2[i]), "crc64")
					So(hash1, ShouldEqual, hash2)
				}
			})
		})
	})
}
