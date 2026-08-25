package util

import (
	"context"
	"fmt"
	"github.com/olekukonko/tablewriter"
	logger "github.com/sirupsen/logrus"
	"github.com/tencentyun/cos-go-sdk-v5"
	"os"
	"strings"
	"time"
)

// PutBucketInventory todo
func PutBucketInventory(c *cos.Client, id, configuration string) error {
	configurationContent, err := GetContent(configuration)
	if err != nil {
		return err
	}
	var opt cos.BucketPutInventoryOptions
	err = ParseContent(configurationContent, &opt, ContentTypeInventory)
	if err != nil {
		return err
	}
	_, err = c.Bucket.PutInventory(context.Background(), id, &opt)
	if err != nil {
		return err
	}
	logger.Info("PutInventory success")
	return err
}

// GetBucketInventory todo
func GetBucketInventory(c *cos.Client, id string) error {
	res, _, err := c.Bucket.GetInventory(context.Background(), id)
	if err != nil {
		return err
	}
	RenderInventoryConfigDetail(res)
	return err
}

// DeleteBucketInventory todo
func DeleteBucketInventory(c *cos.Client, id string) error {
	_, err := c.Bucket.DeleteInventory(context.Background(), id)
	if err != nil {
		return err
	}
	logger.Info("DeleteInventory success")
	return err
}

// PostBucketInventory todo
func PostBucketInventory(c *cos.Client, id, configuration string) error {
	configurationContent, err := GetContent(configuration)
	if err != nil {
		return err
	}
	var opt cos.BucketPostInventoryOptions
	err = ParseContent(configurationContent, &opt, ContentTypeInventory)
	if err != nil {
		return err
	}
	_, err = c.Bucket.PostInventory(context.Background(), id, &opt)
	if err != nil {
		return err
	}
	logger.Info("PostInventory success")
	return err
}

// ListBucketInventory 展示清单配置
func ListBucketInventory(client *cos.Client) error {
	// 创建表格
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Status", "Schedule", "IncludedObjectVersions", "Destination", "Filter", "Fields"})
	table.SetBorder(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoWrapText(false)
	table.SetRowLine(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoMergeCells(false)
	table.SetCaption(true, "Detailed COS Bucket Inventory Configurations")

	// 初始化分页参数
	token := ""
	isTruncated := true
	totalConfigs := 0

	// 分页获取配置
	for isTruncated {
		// 获取当前页配置
		result, _, err := client.Bucket.ListInventoryConfigurations(context.Background(), token)
		if err != nil {
			return fmt.Errorf("failed to list inventory configurations: %w", err)
		}

		// 处理当前页配置
		for _, config := range result.InventoryConfigurations {
			// 添加配置到表格
			table.Append(formatDetailedInventoryRow(config))
			totalConfigs++
		}

		// 更新分页状态
		isTruncated = result.IsTruncated
		token = result.NextContinuationToken
	}

	// 渲染表格
	table.Render()

	// 显示摘要信息
	fmt.Printf("\nTotal inventory configurations: %d\n", totalConfigs)
	return nil
}

// formatDetailedInventoryRow 格式化详细清单配置行
func formatDetailedInventoryRow(config cos.BucketListInventoryConfiguartion) []string {
	// 格式化状态
	status := "Disabled"
	if config.IsEnabled == "true" {
		status = "Enabled"
	}

	// 格式化计划
	schedule := fmt.Sprintf("Frequency: %s", config.Schedule.Frequency)

	// 格式化目标
	cosDest := config.Destination
	destination := fmt.Sprintf("Bucket: %s\nFormat: %s", cosDest.Bucket, cosDest.Format)
	if cosDest.AccountId != "" {
		destination += fmt.Sprintf("\nAccount: %s", cosDest.AccountId)
	}
	if cosDest.Prefix != "" {
		destination += fmt.Sprintf("\nPrefix: %s", cosDest.Prefix)
	}

	// 格式化过滤器
	filter := formatFilter(config.Filter)

	// 格式化字段
	fields := formatDetailedFields(config.OptionalFields)

	return []string{
		config.ID,
		status,
		schedule,
		config.IncludedObjectVersions,
		destination,
		filter,
		fields,
	}
}

// formatFilter 格式化过滤器
func formatFilter(filter *cos.BucketInventoryFilter) string {
	if filter == nil {
		return "No filter"
	}

	var parts []string

	// 前缀过滤
	if filter.Prefix != "" {
		parts = append(parts, fmt.Sprintf("Prefix: %s", filter.Prefix))
	}

	// 标签过滤
	if len(filter.Tags) > 0 {
		tagStrs := make([]string, len(filter.Tags))
		for i, tag := range filter.Tags {
			tagStrs[i] = fmt.Sprintf("%s=%s", tag.Key, tag.Value)
		}
		parts = append(parts, fmt.Sprintf("Tags: %s", strings.Join(tagStrs, ", ")))
	}

	// 存储类型过滤
	if filter.StorageClass != "" {
		parts = append(parts, fmt.Sprintf("StorageClass: %s", filter.StorageClass))
	}

	// 时间范围过滤
	if filter.Period != nil {
		start := time.Unix(filter.Period.StartTime, 0).Format("2006-01-02")
		end := time.Unix(filter.Period.EndTime, 0).Format("2006-01-02")
		parts = append(parts, fmt.Sprintf("Period: %s to %s", start, end))
	}

	if len(parts) == 0 {
		return "No filter"
	}

	return strings.Join(parts, "\n")
}

// formatDetailedFields 格式化详细字段信息
func formatDetailedFields(fields *cos.BucketInventoryOptionalFields) string {
	if fields == nil || len(fields.BucketInventoryFields) == 0 {
		return "No optional fields"
	}

	return strings.Join(fields.BucketInventoryFields, "\n")
}

// RenderInventoryConfigDetail todo
func RenderInventoryConfigDetail(config *cos.BucketGetInventoryResult) {
	// 创建表格
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Section", "Key", "Value"})
	table.SetAutoMergeCells(true)
	table.SetRowLine(true)
	table.SetBorder(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoWrapText(false)
	table.SetCaption(true, "Inventory Configuration Details")

	// 添加基本信息
	table.Append([]string{"Basic", "ID", config.ID})
	table.Append([]string{"Basic", "Enabled", config.IsEnabled})
	table.Append([]string{"Basic", "Included Versions", config.IncludedObjectVersions})

	// 添加分隔行
	table.Append([]string{"", "", ""})

	// 添加计划信息
	if config.Schedule != nil {
		table.Append([]string{"Schedule", "Frequency", config.Schedule.Frequency})
	} else {
		table.Append([]string{"Schedule", "", "Not configured"})
	}

	// 添加分隔行
	table.Append([]string{"", "", ""})

	// 添加目标信息
	if config.Destination != nil {
		dest := config.Destination
		table.Append([]string{"Destination", "Bucket", dest.Bucket})
		table.Append([]string{"Destination", "Format", dest.Format})

		if dest.AccountId != "" {
			table.Append([]string{"Destination", "Account ID", dest.AccountId})
		}

		if dest.Prefix != "" {
			table.Append([]string{"Destination", "Prefix", dest.Prefix})
		}

	} else {
		table.Append([]string{"Destination", "", "Not configured"})
	}

	// 添加分隔行
	table.Append([]string{"", "", ""})

	// 添加过滤器信息
	if config.Filter != nil {
		// 前缀过滤
		if config.Filter.Prefix != "" {
			table.Append([]string{"Filter", "Prefix", config.Filter.Prefix})
		}

		// 标签过滤
		if len(config.Filter.Tags) > 0 {
			for i, tag := range config.Filter.Tags {
				section := "Filter"
				if i > 0 {
					section = ""
				}
				table.Append([]string{section, "Tag", fmt.Sprintf("%s=%s", tag.Key, tag.Value)})
			}
		}

		// 存储类型过滤
		if config.Filter.StorageClass != "" {
			table.Append([]string{"Filter", "Storage Class", config.Filter.StorageClass})
		}

		// 时间范围过滤
		if config.Filter.Period != nil {
			startTime := time.Unix(config.Filter.Period.StartTime, 0).Format("2006-01-02")
			endTime := time.Unix(config.Filter.Period.EndTime, 0).Format("2006-01-02")
			table.Append([]string{"Filter", "Period", fmt.Sprintf("%s to %s", startTime, endTime)})
		}
	} else {
		table.Append([]string{"Filter", "", "Not configured"})
	}

	// 添加分隔行
	table.Append([]string{"", "", ""})

	// 添加可选字段
	if config.OptionalFields != nil && len(config.OptionalFields.BucketInventoryFields) > 0 {
		section := "Optional Fields"
		for _, field := range config.OptionalFields.BucketInventoryFields {
			table.Append([]string{section, "Field", field})
		}
	} else {
		table.Append([]string{"Optional Fields", "", "No fields configured"})
	}

	// 渲染表格
	table.Render()
}
