package util

type Config struct {
	Base    BaseCfg  `yaml:"base"`
	Buckets []Bucket `yaml:"buckets"`
}

type BaseCfg struct {
	SecretID     string `yaml:"secretid"`
	SecretKey    string `yaml:"secretkey"`
	SessionToken string `yaml:"sessiontoken"`
	//AppID     string `yaml:"appid"`
}

type Bucket struct {
	Name   string `yaml:"name"`
	Alias  string `yaml:"alias"`
	Region string `yaml:"region"`
}

type UploadInfo struct {
	Key       string `xml:"Key,omitempty"`
	UploadID  string `xml:"UploadId,omitempty"`
	Initiated string `xml:"Initiated,omitempty"`
}
