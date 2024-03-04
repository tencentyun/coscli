package cmd

import (
	"context"
	"coscli/util"
	"net/http"
	"net/url"
	"os"
	"time"

	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/tencentyun/cos-go-sdk-v5"
)

var signurlCmd = &cobra.Command{
	Use:   "signurl",
	Short: "Gets the signed download URL",
	Long: `Gets the signed download URL

Format:
  ./coscli signurl cos://<bucket-name>/<key> [flags]

Example:
  ./coscli signurl cos://examplebucket/test.jpg -t 100`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		time, _ := cmd.Flags().GetInt("time")
		if util.IsCosPath(args[0]) {
			GetSignedURL(args[0], time)
		} else {
			logger.Fatalln("cospath needs to contain cos://")
		}
	},
}

func init() {
	rootCmd.AddCommand(signurlCmd)

	signurlCmd.Flags().IntP("time", "t", 10000, "Set the validity time of the signature(Default 10000)")
}

func GetSignedURL(path string, t int) {
	bucketName, cosPath := util.ParsePath(path)
	c := util.NewClient(&config, &param, bucketName)

	opt := &cos.PresignedURLOptions{
		Query:  &url.Values{},
		Header: &http.Header{},
	}
	// 格式化参数
	secretID, secretKey, secretToken := config.Base.SecretID, config.Base.SecretKey, config.Base.SessionToken
	if param.SecretID != "" {
		secretID = param.SecretID
		secretToken = ""
	}
	if param.SecretKey != "" {
		secretKey = param.SecretKey
		secretToken = ""
	}
	if param.SessionToken != "" {
		secretToken = param.SessionToken
	}
	if secretToken != "" {
		opt.Query.Add("x-cos-security-token", secretToken)
	}

	presignedURL, err := c.Object.GetPresignedURL(context.Background(), http.MethodGet, cosPath,
		secretID, secretKey, time.Second*time.Duration(t), opt)
	if err != nil {
		logger.Fatalln(err)
		os.Exit(1)
	}

	logger.Infoln("Signed URL:")
	logger.Infoln(presignedURL)
}
