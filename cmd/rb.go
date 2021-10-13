package cmd

import (
	"context"
	"coscli/util"
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var rbCmd = &cobra.Command{
	Use:   "rb",
	Short: "Remove bucket",
	Long:  `Remove bucket

Format:
  ./coscli rb cos://<bucket-name>-<app-id> -r region

Example:
  ./coscli rb cos://example-1234567890 -r ap-shanghai`,
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(1)(cmd, args); err != nil {
			return err
		}
		bucketIDName, cosPath := util.ParsePath(args[0])
		if bucketIDName == "" || cosPath != "" {
			return fmt.Errorf("Invalid arguments! ")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		bucketIDName, _ := util.ParsePath(args[0])
		region, _ := cmd.Flags().GetString("region")

		removeBucket(bucketIDName, region)
	},
}

func init() {
	rootCmd.AddCommand(rbCmd)

	rbCmd.Flags().StringP("region", "r", "", "Region")

	_ = rbCmd.MarkFlagRequired("region")
}

func removeBucket(bucketIDName string, region string) {
	c := util.CreateClient(&config, bucketIDName, region)

	_, err := c.Bucket.Delete(context.Background())
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Delete a empty bucket! name: %s region: %s\n", bucketIDName, region)
}
