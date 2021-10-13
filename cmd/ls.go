package cmd

import (
	"coscli/util"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/tencentyun/cos-go-sdk-v5"
	"os"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List buckets or objects",
	Long:  `List buckets or objects

Format:
  ./coscli ls cos://<bucket-name>[/prefix/] [flags]

Example:
  ./coscli ls cos://examplebucket/test/ -r`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := cmd.Flags().GetInt("limit")
		recursive, _ := cmd.Flags().GetBool("recursive")
		include, _ := cmd.Flags().GetString("include")
		exclude, _ := cmd.Flags().GetString("exclude")
		if limit < 0 || limit > 1000 {
			_, _ = fmt.Fprintln(os.Stderr, "Flag --limit should in range 0~1000")
			os.Exit(1)
		}

		// 无参数，则列出当前账号下的所有存储桶
		if len(args) == 0 {
			listBuckets(limit, include, exclude)
		} else {
			listObjects(args[0], limit, recursive, include, exclude)
		}
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)

	lsCmd.Flags().Int("limit", 0, "Limit the number of objects listed(0~1000)")
	lsCmd.Flags().BoolP("recursive", "r", false, "List objects recursively")
	lsCmd.Flags().String("include", "", "List files that meet the specified criteria")
	lsCmd.Flags().String("exclude", "", "Exclude files that meet the specified criteria")
}

func listBuckets(limit int, include string, exclude string) {
	c := util.NewClient(&config, "")

	buckets := util.GetBucketsList(c, limit, include, exclude)

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Bucket Name", "Region", "Create Date"})
	for _, b := range buckets {
		table.Append([]string{b.Name, b.Region, b.CreationDate})
	}
	table.SetFooter([]string{"", "Total Buckets: ", fmt.Sprintf("%d", len(buckets))})
	table.SetBorder(false)
	table.Render()
}

func listObjects(cosPath string, limit int, recursive bool, include string, exclude string) {
	bucketName, path := util.ParsePath(cosPath)
	c := util.NewClient(&config, bucketName)

	var dirs []string
	var objects []cos.Object
	if recursive {
		objects = util.GetObjectsListRecursive(c, path, limit, include, exclude)
	} else {
		dirs, objects = util.GetObjectsList(c, path, limit, include, exclude)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Key", "Type", "Last Modified", "Size"})
	for _, d := range dirs {
		table.Append([]string{d, "DIR", "", ""})
	}
	for _, v := range objects {
		table.Append([]string{v.Key, v.StorageClass, v.LastModified, util.FormatSize(v.Size)})
	}
	table.SetBorder(false)
	table.SetAlignment(tablewriter.ALIGN_RIGHT)
	table.SetFooter([]string{"", "", "Total Objects: ", fmt.Sprintf("%d", len(dirs) + len(objects))})
	table.Render()
}
