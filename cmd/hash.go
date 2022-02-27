package cmd

import (
	"coscli/util"
	"os"

	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var hashCmd = &cobra.Command{
	Use:   "hash",
	Short: "Calculate local file's hash-code or show cos file's hash-code",
	Long: `Calculate local file's hash-code or show cos file's hash-code

Format:
  ./coscli hash <file-path> [--type <hash-type>]

Example:
  ./coscli hash cos://example --type md5`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		bucketName, path := util.ParsePath(args[0])
		hashType, _ := cmd.Flags().GetString("type")

		if bucketName != "" {
			showHash(bucketName, path, hashType)
		} else {
			calculateHash(path, hashType)
		}
	},
}

func init() {
	rootCmd.AddCommand(hashCmd)

	hashCmd.Flags().StringP("type", "", "crc64", "Choose the hash type(md5 or crc64)")
}

func showHash(bucketName string, path string, hashType string) {
	c := util.NewClient(&config, &param, bucketName)
	switch hashType {
	case "crc64":
		h, _ := util.ShowHash(c, path, "crc64")
		logger.Infoln("crc64-ecma:  ", h)
	case "md5":
		h, b := util.ShowHash(c, path, "md5")
		logger.Infoln("md5:    ", h)
		logger.Infoln("base64: ", b)
	default:
		logger.Infoln("Wrong args!")
	}
}

func calculateHash(path string, hashType string) (h string) {
	switch hashType {
	case "crc64":
		h, _ := util.CalculateHash(path, "crc64")
		logger.Infoln("crc64-ecma:  ", h)
	case "md5":
		f, err := os.Stat(path)
		if err != nil {
			logger.Fatalln(err)
			os.Exit(1)
		}

		if (float64(f.Size()) / 1024 / 1024) > 32 {
			logger.Fatalln("MD5 of large files is not supported")
			os.Exit(1)
		}

		h, b := util.CalculateHash(path, "md5")
		logger.Infof("md5:     %s\n", h)
		logger.Infoln("base64: ", b)
	default:
		logger.Infoln("Wrong args!")
	}
	return h
}
