package cmd

import (
	"coscli/util"
	"fmt"
	"github.com/spf13/cobra"
)

var inventoryCmd = &cobra.Command{
	Use:   "inventory",
	Short: "Modify inventory",
	Long: `Modify inventory

Format:
	./coscli inventory --method [method] cos://<bucket-name> 

Example:
	./coscli inventory --method put cos://examplebucket --task-id list4 --configuration "<?xml version=\"1.0\" encoding=\"UTF-8\"?><InventoryConfiguration xmlns=\"http://....\"><Id>list4</Id><IsEnabled>false</IsEnabled><Destination><COSBucketDestination><Format>CSV</Format><AccountId>100000000002</AccountId><Bucket>qcs::cos:ap-nanjing::test-1000000001</Bucket><Prefix>list4</Prefix><Encryption><SSE-COS></SSE-COS></Encryption></COSBucketDestination></Destination><Schedule><Frequency>Weekly</Frequency></Schedule><Filter><And><Prefix>myPrefix</Prefix><Tag><Key>age</Key><Value>18</Value></Tag></And><Period><StartTime>1768688761</StartTime><EndTime>1568688762</EndTime></Period></Filter><IncludedObjectVersions>All</IncludedObjectVersions><OptionalFields><Field>Size</Field><Field>Tag</Field><Field>LastModifiedDate</Field><Field>ETag</Field><Field>StorageClass</Field><Field>IsMultipartUploaded</Field></OptionalFields></InventoryConfiguration>" 
	./coscli inventory --method get cos://examplebucket --task-id list4
	./coscli inventory --method list cos://examplebucket
	./coscli inventory --method delete cos://examplebucket --task-id list4
	./coscli inventory --method post cos://examplebucket --task-id list4 --configuration "<?xml version=\"1.0\" encoding=\"UTF-8\"?><InventoryConfiguration xmlns=\"http://....\"><Id>list4</Id><Destination><COSBucketDestination><Format>CSV</Format><AccountId>100000000002</AccountId><Bucket>qcs::cos:ap-nanjing::test-100000001</Bucket><Prefix>list4</Prefix><Encryption><SSE-COS>111</SSE-COS></Encryption></COSBucketDestination></Destination><Filter><And><Prefix>myPrefix</Prefix><Tag><Key>age</Key><Value>18</Value></Tag></And><Period><StartTime>1768688761</StartTime><EndTime>1568688762</EndTime></Period></Filter><IncludedObjectVersions>All</IncludedObjectVersions><OptionalFields><Field>Size</Field><Field>Tag</Field><Field>LastModifiedDate</Field><Field>ETag</Field><Field>StorageClass</Field><Field>IsMultipartUploaded</Field></OptionalFields></InventoryConfiguration>"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		method, _ := cmd.Flags().GetString("method")
		taskId, _ := cmd.Flags().GetString("task-id")
		configuration, _ := cmd.Flags().GetString("configuration")

		cosUrl, err := util.FormatUrl(args[0])
		if err != nil {
			return fmt.Errorf("cos url format error:%v", err)
		}

		if !cosUrl.IsCosUrl() {
			return fmt.Errorf("cospath needs to contain cos://")
		}

		bucketName := cosUrl.(*util.CosUrl).Bucket

		c, err := util.NewClient(&config, &param, bucketName)
		if err != nil {
			return err
		}

		if method == "put" {
			if taskId == "" {
				return fmt.Errorf("--task-id is required when --method is 'put'")
			}
			err = util.PutBucketInventory(c, taskId, configuration)
		} else if method == "get" {
			// 显式要求用户传 --task-id；如需查看清单列表请使用 --method list。
			// 否则 SDK 会把空 id 拼成 ?inventory 列表请求，但客户端按
			// <InventoryConfiguration> 解析 → 报 XML schema 不匹配。
			if taskId == "" {
				return fmt.Errorf("--task-id is required when --method is 'get'; use '--method list' to list all inventory configurations")
			}
			err = util.GetBucketInventory(c, taskId)
		} else if method == "list" {
			err = util.ListBucketInventory(c)
		} else if method == "delete" {
			if taskId == "" {
				return fmt.Errorf("--task-id is required when --method is 'delete'")
			}
			err = util.DeleteBucketInventory(c, taskId)
		} else if method == "post" {
			if taskId == "" {
				return fmt.Errorf("--task-id is required when --method is 'post'")
			}
			err = util.PostBucketInventory(c, taskId, configuration)
		} else {
			err = fmt.Errorf("method '%s' is not supported, valid methods are 'put', 'get', 'list', 'delete', and 'post'", method)
		}

		return err
	},
}

func init() {
	rootCmd.AddCommand(inventoryCmd)
	inventoryCmd.Flags().String("method", "", "put/get")
	inventoryCmd.Flags().String("task-id", "", "The name of the checklist task.Default value: None.Valid characters: a-z, A-Z, 0-9, -, _, .")
	inventoryCmd.Flags().String("configuration", "", "Store checklist configuration information, supporting both XML and JSON syntax; if the input contains the file:// prefix, it indicates reading the configuration from a file.")
}
