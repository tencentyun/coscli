package cmd

import (
	"context"
	"coscli/util"
	"fmt"
	"net/http"
	"net/url"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		time, _ := cmd.Flags().GetInt("time")
		var err error
		if util.IsCosPath(args[0]) {
			err = GetSignedURL(args[0], time)
		} else {
			return fmt.Errorf("cospath needs to contain cos://")
		}

		return err
	},
}

func init() {
	rootCmd.AddCommand(signurlCmd)

	signurlCmd.Flags().IntP("time", "t", 10000, "Set the validity time of the signature(Default 10000)")
}

func GetSignedURL(path string, t int) error {
	bucketName, cosPath := util.ParsePath(path)
	c, err := util.NewClient(&config, &param, bucketName)
	if err != nil {
		return err
	}

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
		return err
	}

	logger.Infoln("Signed URL:")
	logger.Infoln(presignedURL)

	return nil
}
