package cmd

import (
	"context"
	"coscli/util"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/tencentyun/cos-go-sdk-v5"
)

var bucketTaggingCmd = &cobra.Command{
	Use:   "bucket-tagging",
	Short: "Modify bucket tagging",
	Long: `Modify bucket tagging

Format:
	./coscli bucket-tagging --method [method] cos://<bucket-name> [tag_key]#[tag_value]

Example:
	./coscli bucket-tagging --method put cos://examplebucket tag1#test1 tag2#test2
	./coscli bucket-tagging --method get cos://examplebucket
	./coscli bucket-tagging --method delete cos://examplebucket`,
	Run: func(cmd *cobra.Command, args []string) {
		method, _ := cmd.Flags().GetString("method")

		if method == "put" {
			if len(args) < 2 {
				logger.Fatalln("not enough arguments in call to put bucket tagging")
			}
			putBucketTagging(args[0], args[1:])
		}

		if method == "get" {
			if len(args) < 1 {
				logger.Fatalln("not enough arguments in call to get bucket tagging")
				os.Exit(1)
			}
			getBucketTagging(args[0])
		}

		if method == "delete" {
			if len(args) < 1 {
				logger.Fatalln("not enough arguments in call to get bucket tagging")
				os.Exit(1)
			}
			deleteBucketTagging(args[0])
		}
	},
}

func init() {
	rootCmd.AddCommand(bucketTaggingCmd)
	bucketTaggingCmd.Flags().String("method", "", "put/get/delete")
}

func putBucketTagging(cosPath string, tags []string) {
	bucketName, _ := util.ParsePath(cosPath)
	c := util.NewClient(&config, &param, bucketName)
	tg := &cos.BucketPutTaggingOptions{}
	for i := 0; i < len(tags); i += 1 {
		tmp := strings.Split(tags[i], "#")
		if len(tmp) >= 2 {
			tg.TagSet = append(tg.TagSet, cos.BucketTaggingTag{Key: tmp[0], Value: tmp[1]})
		} else {
			logger.Fatalln("invalid tag")
			os.Exit(1)
		}
	}

	_, err := c.Bucket.PutTagging(context.Background(), tg)
	if err != nil {
		logger.Infoln(err.Error())
		logger.Fatalln(err)
		os.Exit(1)
	}
}

func getBucketTagging(cosPath string) {
	bucketName, _ := util.ParsePath(cosPath)
	c := util.NewClient(&config, &param, bucketName)

	v, _, err := c.Bucket.GetTagging(context.Background())
	if err != nil {
		logger.Infoln(err.Error())
		logger.Fatalln(err)
		os.Exit(1)
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Key", "Value"})
	for _, t := range v.TagSet {
		table.Append([]string{t.Key, t.Value})
	}
	table.SetBorder(false)
	table.SetAlignment(tablewriter.ALIGN_RIGHT)
	table.Render()
}

func deleteBucketTagging(cosPath string) {
	bucketName, _ := util.ParsePath(cosPath)
	c := util.NewClient(&config, &param, bucketName)

	_, err := c.Bucket.DeleteTagging(context.Background())
	if err != nil {
		logger.Infoln(err.Error())
		logger.Fatalln(err)
		os.Exit(1)
	}
}
