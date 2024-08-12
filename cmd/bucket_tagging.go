package cmd

import (
	"context"
	"coscli/util"
	"fmt"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
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
	./coscli bucket-tagging --method delete cos://examplebucket
	./coscli bucket-tagging --method delete cos://examplebucket tag1#test1 tag2#test2`,
	RunE: func(cmd *cobra.Command, args []string) error {
		method, _ := cmd.Flags().GetString("method")

		var err error
		if method == "put" {
			if len(args) < 2 {
				return fmt.Errorf("not enough arguments in call to put bucket tagging")
			}
			err = putBucketTagging(args[0], args[1:])
		}

		if method == "get" {
			if len(args) < 1 {
				return fmt.Errorf("not enough arguments in call to get bucket tagging")
			}
			err = getBucketTagging(args[0])
		}

		if method == "delete" {
			if len(args) < 1 {
				return fmt.Errorf("not enough arguments in call to delete bucket tagging")
			} else if len(args) == 1 {
				err = deleteBucketTagging(args[0])
			} else {
				err = deleteDesBucketTagging(args[0], args[1:])
			}
		}

		return err
	},
}

func init() {
	rootCmd.AddCommand(bucketTaggingCmd)
	bucketTaggingCmd.Flags().String("method", "", "put/get/delete")
}

func putBucketTagging(cosPath string, tags []string) error {
	bucketName, _ := util.ParsePath(cosPath)
	c, err := util.NewClient(&config, &param, bucketName)
	if err != nil {
		return err
	}
	tg := &cos.BucketPutTaggingOptions{}
	for i := 0; i < len(tags); i += 1 {
		tmp := strings.Split(tags[i], "#")
		if len(tmp) >= 2 {
			tg.TagSet = append(tg.TagSet, cos.BucketTaggingTag{Key: tmp[0], Value: tmp[1]})
		} else {
			return fmt.Errorf("invalid tag")
		}
	}

	_, err = c.Bucket.PutTagging(context.Background(), tg)
	if err != nil {
		return err
	}

	return nil
}

func getBucketTagging(cosPath string) error {
	bucketName, _ := util.ParsePath(cosPath)
	c, err := util.NewClient(&config, &param, bucketName)
	if err != nil {
		return err
	}

	v, _, err := c.Bucket.GetTagging(context.Background())
	if err != nil {
		return err
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Key", "Value"})
	for _, t := range v.TagSet {
		table.Append([]string{t.Key, t.Value})
	}
	table.SetBorder(false)
	table.SetAlignment(tablewriter.ALIGN_RIGHT)
	table.Render()

	return nil
}

func deleteBucketTagging(cosPath string) error {
	bucketName, _ := util.ParsePath(cosPath)
	c, err := util.NewClient(&config, &param, bucketName)
	if err != nil {
		return err
	}

	_, err = c.Bucket.DeleteTagging(context.Background())
	if err != nil {
		return err
	}
	return nil
}

func deleteDesBucketTagging(cosPath string, tags []string) error {
	bucketName, _ := util.ParsePath(cosPath)
	c, err := util.NewClient(&config, &param, bucketName)
	if err != nil {
		return err
	}
	d, _, err := c.Bucket.GetTagging(context.Background())
	if err != nil {
		return err
	}
	table := make(map[string]string)
	for _, t := range d.TagSet {
		table[t.Key] = t.Value
	}
	var del []string
	for i := 0; i < len(tags); i += 1 {
		tmp := strings.Split(tags[i], "#")
		if len(tmp) >= 2 {
			_, ok := table[tmp[0]]
			del = append(del, tmp[0])
			if ok {
				delete(table, tmp[0])
			} else {
				return fmt.Errorf("the BucketTagging %s is not exist", tmp[0])
			}
		} else {
			return fmt.Errorf("invalid tag")
		}
	}
	tg := &cos.BucketPutTaggingOptions{}
	for a, b := range table {
		tg.TagSet = append(tg.TagSet, cos.BucketTaggingTag{Key: a, Value: b})
	}

	_, err = c.Bucket.PutTagging(context.Background(), tg)
	if err != nil {
		return err
	}
	fmt.Print("delete BucketTagging ")
	for _, x := range del {
		fmt.Printf("%s ", x)
	}
	fmt.Print("success")
	return nil
}
