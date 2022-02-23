package cmd

import (
	"context"
	"coscli/util"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

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

		GetSignedURL(args[0], time)
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
	if config.Base.SessionToken != "" {
		opt.Query.Add("x-cos-security-token", config.Base.SessionToken)
	}

	presignedURL, err := c.Object.GetPresignedURL(context.Background(), http.MethodGet, cosPath,
		config.Base.SecretID, config.Base.SecretKey, time.Second*time.Duration(t), opt)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println("Signed URL:")
	fmt.Println(presignedURL)
}
