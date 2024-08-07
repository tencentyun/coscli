package util

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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

func CamAuth(roleName string) (data Data, err error) {
	if roleName == "" {
		return data, fmt.Errorf("Get cam auth error : roleName not set")
	}

	// 创建HTTP客户端
	client := &http.Client{}

	// 创建一个5秒的超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 创建一个HTTP GET请求并将上下文与其关联
	req, err := http.NewRequest("GET", CamUrl+roleName, nil)
	if err != nil {
		return data, fmt.Errorf("Get cam auth error : create request error[%v]", err)
	}
	req = req.WithContext(ctx)

	// 发起HTTP GET请求
	res, err := client.Do(req)
	if err != nil {
		// 检查是否超时错误
		if ctx.Err() == context.DeadlineExceeded {
			return data, fmt.Errorf("Get cam auth timeout[%v]", ctx.Err())
		} else {
			return data, fmt.Errorf("Get cam auth error : request error[%v]", err)
		}
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return data, fmt.Errorf("Get cam auth error : get response error[%v]", err)
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return data, fmt.Errorf("Get cam auth error : auth error[%v]", err)
	}

	if data.Code != "Success" {
		return data, fmt.Errorf("Get cam auth error : response error[%v]", err)
	}

	return data, nil
}
