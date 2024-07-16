package cmd

import (
	"coscli/util"
	"fmt"
	"os"
	"strings"

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
	RunE: func(cmd *cobra.Command, args []string) error {
		bucketName, path := util.ParsePath(args[0])
		hashType, _ := cmd.Flags().GetString("type")
		hashType = strings.ToLower(hashType)
		var err error
		if bucketName != "" {
			err = showHash(bucketName, path, hashType)
		} else {
			_, err = calculateHash(path, hashType)
		}

		return err
	},
}

func init() {
	rootCmd.AddCommand(hashCmd)

	hashCmd.Flags().StringP("type", "", "crc64", "Choose the hash type(md5 or crc64)")
}

func showHash(bucketName string, path string, hashType string) error {
	c, err := util.NewClient(&config, &param, bucketName)
	if err != nil {
		return err
	}
	switch hashType {
	case "crc64":
		h, _, _, err := util.ShowHash(c, path, "crc64")
		if err != nil {
			return err
		}
		logger.Infoln("crc64-ecma:  ", h)
	case "md5":
		h, b, _, err := util.ShowHash(c, path, "md5")
		if err != nil {
			return err
		}
		logger.Infoln("md5:    ", h)
		logger.Infoln("base64: ", b)
	default:
		return fmt.Errorf("--type can only be selected between MD5 and CRC64")
	}
	return nil
}

func calculateHash(path string, hashType string) (h string, err error) {
	switch hashType {
	case "crc64":
		h, _, err = util.CalculateHash(path, "crc64")
		if err != nil {
			return "", err
		}
		logger.Infoln("crc64-ecma:  ", h)
	case "md5":
		f, err := os.Stat(path)
		if err != nil {
			return "", err
		}

		if (float64(f.Size()) / 1024 / 1024) > 32 {
			return "", fmt.Errorf("MD5 of large files is not supported")
		}

		h, b, err := util.CalculateHash(path, "md5")
		if err != nil {
			return "", err
		}
		logger.Infof("md5:     %s\n", h)
		logger.Infoln("base64: ", b)
	default:
		return "", fmt.Errorf("--type can only be selected between MD5 and CRC64")
	}
	return h, err
}
