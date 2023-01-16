package cmd

import (
	"fmt"
	"os"

	"coscli/util"

	"github.com/olekukonko/tablewriter"
	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/tencentyun/cos-go-sdk-v5"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List buckets or objects",
	Long: `List buckets or objects

Format:
  ./coscli ls cos://<bucket-name>[/prefix/] [flags]

Example:
  ./coscli ls cos://examplebucket/test/ -r`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := cmd.Flags().GetInt("limit")
		recursive, _ := cmd.Flags().GetBool("recursive")
		include, _ := cmd.Flags().GetString("include")
		exclude, _ := cmd.Flags().GetString("exclude")
		if limit < 0 || limit > 1000 {
			logger.Fatalln("Flag --limit should in range 0~1000")
			os.Exit(1)
		}

		// 无参数，则列出当前账号下的所有存储桶
		if len(args) == 0 {
			listBuckets(limit, include, exclude)
		} else if util.IsCosPath(args[0]) {
			listObjects(args[0], limit, recursive, include, exclude)
		} else {
			logger.Fatalln("cospath needs to contain cos://")
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
	c := util.NewClient(&config, &param, "")

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
	c := util.NewClient(&config, &param, bucketName)
	var dirs []string
	var objects []cos.Object
	var marker = ""
	var isTruncated bool
	var commonPrefixes []string
	var nextMarker string
	var total int64
	var output_num int64
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Key", "Type", "Last Modified", "Size"})
	table.SetBorder(false)
	table.SetAlignment(tablewriter.ALIGN_RIGHT)
	if recursive {
		for {
			output_num = 0
			table.ClearRows()
			objects, isTruncated, nextMarker, commonPrefixes = util.GetObjectsListRecursiveForLs(c, path, limit, include, exclude, marker)
			for _, d := range dirs {
				table.Append([]string{d, "DIR", "", ""})
			}
			for _, v := range objects {
				table.Append([]string{v.Key, v.StorageClass, v.LastModified, util.FormatSize(v.Size)})
			}
			total = total + int64(len(dirs)) + int64(len(objects))
			output_num += int64(len(dirs)) + int64(len(objects))
			if len(commonPrefixes) > 0 {
				for _, v := range commonPrefixes {
					objects, isTruncated, nextMarker, _ = util.GetObjectsListRecursiveForLs(c, v, limit,
						include, exclude, marker)
					for _, d := range dirs {
						table.Append([]string{d, "DIR", "", ""})
					}
					for _, v := range objects {
						table.Append([]string{v.Key, v.StorageClass, v.LastModified, util.FormatSize(v.Size)})
					}
					total = total + int64(len(dirs)) + int64(len(objects))
					output_num += int64(len(dirs)) + int64(len(objects))
				}
			}
			if output_num > 0 {
				table.Render()
				table.ClearRows()
			}
			if !isTruncated {
				break
			}
			marker = nextMarker
		}
		table.SetFooter([]string{"", "", "Total Objects: ", fmt.Sprintf("%d", total)})
		table.Render()
	} else {
		for {
			table.ClearRows()
			dirs, objects, isTruncated, nextMarker = util.GetObjectsListForLs(c, path, limit, include, exclude, marker)
			for _, d := range dirs {
				table.Append([]string{d, "DIR", "", ""})
			}
			for _, v := range objects {
				table.Append([]string{v.Key, v.StorageClass, v.LastModified, util.FormatSize(v.Size)})
			}
			total = total + int64(len(dirs)) + int64(len(objects))
			output_num += int64(len(dirs)) + int64(len(objects))
			if output_num > 0 {
				table.Render()
				table.ClearRows()
			}
			if !isTruncated {
				break
			}
			marker = nextMarker
		}

		table.SetFooter([]string{"", "", "Total Objects: ", fmt.Sprintf("%d", total)})
		table.Render()
	}
}
