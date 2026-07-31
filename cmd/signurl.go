package cmd

import (
	"context"
	"coscli/util"
	"fmt"
	logger "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tencentyun/cos-go-sdk-v5"
)

var signurlCmd = &cobra.Command{
	Use:   "signurl",
	Short: "Gets the signed URL for upload or download",
	Long: `Gets the signed URL for upload or download

Format:
  ./coscli signurl cos://<bucket-name>/<key> [flags]

Example:
  ./coscli signurl cos://examplebucket/test.jpg -t 100
  ./coscli signurl cos://examplebucket/test.jpg -t 3600 --method PUT`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		t, _ := cmd.Flags().GetInt("time")
		simpleOutput, _ := cmd.Flags().GetBool("simple-output")
		method, _ := cmd.Flags().GetString("method")
		var err error
		if util.IsCosPath(args[0]) {
			err = GetSignedURL(args[0], t, simpleOutput, method)
		} else {
			return fmt.Errorf("cospath needs to contain cos://")
		}

		return err
	},
}

func init() {
	rootCmd.AddCommand(signurlCmd)

	signurlCmd.Flags().IntP("time", "t", 10000, "Set the validity time of the signature(Default 10000)")
	signurlCmd.Flags().BoolP("simple-output", "", false, "Set simple output mode")
	signurlCmd.Flags().StringP("method", "m", "GET", "Set the HTTP method for the signed URL (GET for download, PUT for upload)")
}

// GetSignedURL 生成签名url
func GetSignedURL(path string, t int, simpleOutput bool, method string) error {
	bucketName, cosPath := util.ParsePath(path)
	c, err := util.NewClient(&config, &param, bucketName)
	if err != nil {
		return err
	}

	// 校验 method 参数
	method = strings.ToUpper(method)
	if method != http.MethodGet && method != http.MethodPut {
		return fmt.Errorf("unsupported method: %s, only GET and PUT are supported", method)
	}

	opt := &cos.PresignedURLOptions{
		Query:  &url.Values{},
		Header: &http.Header{},
	}

	presignedURL, err := c.Object.GetPresignedURL2(context.Background(), method, cosPath, time.Second*time.Duration(t), opt)
	if err != nil {
		return err
	}

	if simpleOutput {
		fmt.Println(presignedURL)
	} else {
		logger.Infoln("Signed URL:")
		logger.Infoln(presignedURL)
	}

	return nil
}
