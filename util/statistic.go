package util

import (
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	logger "github.com/sirupsen/logrus"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func Statistic(objects []cos.Object) {
	var standardCnt, standardIACnt, intelligentTieringCnt, archiveCnt, deepArchiveCnt int
	var mazStandardCnt, mazStandardIACnt, mazIntelligentTieringCnt, mazArchiveCnt int
	var standardSize, standardIASize, intelligentTieringSize, archiveSize, deepArchiveSize int64
	var mazStandardSize, mazStandardIASize, mazIntelligentTieringSize, mazArchiveSize int64

	var totalCnt int
	var totalSize int64

	for _, o := range objects {
		switch o.StorageClass {
		case Standard:
			standardCnt++
			standardSize += o.Size
			totalSize += o.Size
		case StandardIA:
			standardIACnt++
			standardIASize += o.Size
			totalSize += o.Size
		case IntelligentTiering:
			intelligentTieringCnt++
			intelligentTieringSize += o.Size
			totalSize += o.Size
		case Archive:
			archiveCnt++
			archiveSize += o.Size
			totalSize += o.Size
		case DeepArchive:
			deepArchiveCnt++
			deepArchiveSize += o.Size
			totalSize += o.Size
		case MAZStandard:
			mazStandardCnt++
			mazStandardSize += o.Size
			totalSize += o.Size
		case MAZStandardIA:
			mazStandardIACnt++
			mazStandardIASize += o.Size
			totalSize += o.Size
		case MAZIntelligentTiering:
			mazIntelligentTieringCnt++
			mazIntelligentTieringSize += o.Size
			totalSize += o.Size
		case MAZArchive:
			mazArchiveCnt++
			mazArchiveSize += o.Size
			totalSize += o.Size
		}
	}
	totalCnt = len(objects)

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
