package util

import (
	"net/url"
	"testing"

	"bou.ke/monkey"
	"github.com/stretchr/testify/assert"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestNewClient_Cvm(t *testing.T) {
	data := &Data{}
	guard := monkey.Patch(CamAuth, func(string) Data {
		return *data
	})
	defer guard.Unpatch()
	monkey.Patch(GenURL, func(*Config, *Param, string) *cos.BaseURL {
		return nil
	})
	config := &Config{
		Base: BaseCfg{
			Mode: "CvmRole",
		},
	}
	param := &Param{}
	res, _ := NewClient(config, param, "")
	got := res.BaseURL.BucketURL
	assert.Equal(t, got, (*url.URL)(nil), "they should be equal")
}

func TestCreateClient_Cvm(t *testing.T) {
	data := &Data{}
	guard := monkey.Patch(CamAuth, func(string) Data {
		return *data
	})
	defer guard.Unpatch()
	monkey.Patch(CreateURL, func(string, string, string, bool) *cos.BaseURL {
		return nil
	})
	config := &Config{
		Base: BaseCfg{
			Mode: "CvmRole",
		},
	}
	param := &Param{}
	res, _ := CreateClient(config, param, "")
	got := res.BaseURL.BucketURL
	assert.Equal(t, got, (*url.URL)(nil), "they should be equal")
}

func TestNewClient(t *testing.T) {
	config := &Config{
		Base: BaseCfg{
			CloseAutoSwitchHost: "true",
		},
	}
	param := &Param{
		SecretID:     "123",
		SecretKey:    "123",
		SessionToken: "123",
		Protocol:     "test",
	}
	guard := monkey.Patch(GenBaseURL, func(*Config, *Param) *cos.BaseURL {
		return nil
	})
	defer guard.Unpatch()
	res, _ := NewClient(config, param, "")
	got := res.BaseURL.BucketURL
	assert.Equal(t, got, (*url.URL)(nil), "they should be equal")
}

func TestCreateClient(t *testing.T) {
	config := &Config{
		Base: BaseCfg{
			CloseAutoSwitchHost: "true",
			Protocol:            "test",
		},
	}
	param := &Param{
		SecretID:     "123",
		SecretKey:    "123",
		SessionToken: "123",
		Protocol:     "test",
	}
	guard := monkey.Patch(CreateURL, func(string, string, string, bool) *cos.BaseURL {
		return nil
	})
	defer guard.Unpatch()
	res, _ := CreateClient(config, param, "")
	got := res.BaseURL.BucketURL
	assert.Equal(t, got, (*url.URL)(nil), "they should be equal")
}
