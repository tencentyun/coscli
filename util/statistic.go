package util

import (
	"context"
	"fmt"
	"github.com/olekukonko/tablewriter"
	logger "github.com/sirupsen/logrus"
	"github.com/tencentyun/cos-go-sdk-v5"
	"net/url"
	"os"
)

var standardCnt, standardIACnt, intelligentTieringCnt, archiveCnt, deepArchiveCnt int
var mazStandardCnt, mazStandardIACnt, mazIntelligentTieringCnt, mazArchiveCnt int
var standardSize, standardIASize, intelligentTieringSize, archiveSize, deepArchiveSize int64
var mazStandardSize, mazStandardIASize, mazIntelligentTieringSize, mazArchiveSize int64

var totalCnt int
var totalSize int64

func DuObjects(c *cos.Client, cosUrl StorageUrl, filters []FilterOptionType) error {
	// 根据s.Header判断是否是融合桶或者普通桶
	s, err := c.Bucket.Head(context.Background())
	if err != nil {
		return err
	}

	if s.Header.Get("X-Cos-Bucket-Arch") == "OFS" {
		prefix := cosUrl.(*CosUrl).Object
		err = countOfsObjects(c, prefix, filters, "")
	} else {
		err = countCosObjects(c, cosUrl, filters)
	}

	if err != nil {
		return err
	}

	// 输出最终统计数据
	printStatistic()
	return nil
}

func countCosObjects(c *cos.Client, cosUrl StorageUrl, filters []FilterOptionType) error {
	var err error
	var objects []cos.Object
	marker := ""
	isTruncated := true

	for isTruncated {

		err, objects, _, isTruncated, marker = getCosObjectListForLs(c, cosUrl, marker, 0, true)
		if err != nil {
			return fmt.Errorf("list objects error : %v", err)
		}

		for _, object := range objects {
			object.Key, _ = url.QueryUnescape(object.Key)
			if cosObjectMatchPatterns(object.Key, filters) {
				statisticObjects(object)
			}

		}
	}

	return nil
}

func countOfsObjects(c *cos.Client, prefix string, filters []FilterOptionType, marker string) error {
	var err error
	var objects []cos.Object
	var commonPrefixes []string
	isTruncated := true

	for isTruncated {
		err, objects, commonPrefixes, isTruncated, marker = getOfsObjectListForLs(c, prefix, marker, 0, true)
		if err != nil {
			return fmt.Errorf("list objects error : %v", err)
		}

		for _, object := range objects {
			object.Key, _ = url.QueryUnescape(object.Key)
			if cosObjectMatchPatterns(object.Key, filters) {
				statisticObjects(object)
			}
		}

		if len(commonPrefixes) > 0 {
			for _, commonPrefix := range commonPrefixes {
				commonPrefix, _ = url.QueryUnescape(commonPrefix)
				// 递归目录
				err = countOfsObjects(c, commonPrefix, filters, "")
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// 统计对象数
func statisticObjects(object cos.Object) {
	switch object.StorageClass {
	case Standard:
		standardCnt++
		standardSize += object.Size
		totalSize += object.Size
	case StandardIA:
		standardIACnt++
		standardIASize += object.Size
		totalSize += object.Size
	case IntelligentTiering:
		intelligentTieringCnt++
		intelligentTieringSize += object.Size
		totalSize += object.Size
	case Archive:
		archiveCnt++
		archiveSize += object.Size
		totalSize += object.Size
	case DeepArchive:
		deepArchiveCnt++
		deepArchiveSize += object.Size
		totalSize += object.Size
	case MAZStandard:
		mazStandardCnt++
		mazStandardSize += object.Size
		totalSize += object.Size
	case MAZStandardIA:
		mazStandardIACnt++
		mazStandardIASize += object.Size
		totalSize += object.Size
	case MAZIntelligentTiering:
		mazIntelligentTieringCnt++
		mazIntelligentTieringSize += object.Size
		totalSize += object.Size
	case MAZArchive:
		mazArchiveCnt++
		mazArchiveSize += object.Size
		totalSize += object.Size
	}

	totalCnt += 1
}

func printStatistic() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Storage Class", "Objects Count", "Total Size"})
	table.Append([]string{Standard, fmt.Sprintf("%d", standardCnt), FormatSize(standardSize)})
	table.Append([]string{StandardIA, fmt.Sprintf("%d", standardIACnt), FormatSize(standardIASize)})
	table.Append([]string{IntelligentTiering, fmt.Sprintf("%d", intelligentTieringCnt), FormatSize(intelligentTieringSize)})
	table.Append([]string{Archive, fmt.Sprintf("%d", archiveCnt), FormatSize(archiveSize)})
	table.Append([]string{DeepArchive, fmt.Sprintf("%d", deepArchiveCnt), FormatSize(deepArchiveSize)})
	table.Append([]string{MAZStandard, fmt.Sprintf("%d", mazStandardCnt), FormatSize(mazStandardSize)})
	table.Append([]string{MAZStandardIA, fmt.Sprintf("%d", mazStandardIACnt), FormatSize(mazStandardIASize)})
	table.Append([]string{MAZIntelligentTiering, fmt.Sprintf("%d", mazIntelligentTieringCnt), FormatSize(mazIntelligentTieringSize)})
	table.Append([]string{MAZArchive, fmt.Sprintf("%d", mazArchiveCnt), FormatSize(mazArchiveSize)})

	table.SetAlignment(tablewriter.ALIGN_RIGHT)
	table.SetBorders(tablewriter.Border{
		Left:   false,
		Right:  false,
		Top:    false,
		Bottom: true,
	})
	table.Render()
	logger.Infof("Total Objects Count: %d\n", totalCnt)
	logger.Infof("Total Objects Size:  %s\n", FormatSize(totalSize))
}
