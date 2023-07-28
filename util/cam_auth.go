package util

import (
	"context"
	"encoding/json"
	logger "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

const (
	CamUrl = "http://metadata.tencentyun.com/meta-data/cam/security-credentials/"
)

type Data struct {
	TmpSecretId  string `json:"TmpSecretId"`
	TmpSecretKey string `json:"TmpSecretKey"`
	ExpiredTime  int    `json:"ExpiredTime"`
	Expiration   string `json:"Expiration"`
	Token        string `json:"Token"`
	Code         string `json:"Code"`
}

var data Data

func CamAuth(roleName string) Data {
	if roleName == "" {
		logger.Fatalln("Get cam auth error : roleName not set")
		os.Exit(1)
	}

	// 创建HTTP客户端
	client := &http.Client{}

	// 创建一个5秒的超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 创建一个HTTP GET请求并将上下文与其关联
	req, err := http.NewRequest("GET", CamUrl+roleName, nil)
	if err != nil {
		logger.Fatalln("Get cam auth error : create request error", err)
		os.Exit(1)
	}
	req = req.WithContext(ctx)

	// 发起HTTP GET请求
	res, err := client.Do(req)
	if err != nil {
		// 检查是否超时错误
		if ctx.Err() == context.DeadlineExceeded {
			logger.Fatalln("Get cam auth timeout", ctx.Err())
		} else {
			logger.Fatalln("Get cam auth error : request error", err)
		}
		os.Exit(1)
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logger.Fatalln("Get cam auth error : get response error", err)
		os.Exit(1)
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		logger.Fatalln("Get cam auth error : unmarshal json error", err)
		os.Exit(1)
	}

	if data.Code != "Success" {
		logger.Fatalln("Get cam auth error : response code error", err)
		os.Exit(1)
	}

	return data
}
