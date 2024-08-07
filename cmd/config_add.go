package cmd

import (
	"fmt"
	"github.com/mitchellh/go-homedir"
	"os"

	"coscli/util"

	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Used to add a new bucket configuration",
	Long: `Used to add a new bucket configuration

Format:
  ./coscli config add -b <bucket-name> -e <endpoint> -a <alias> [-c <config-file-path>]

Example:
  ./coscli config add -b example-1234567890 -r ap-shanghai -a example`,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := addBucketConfig(cmd)
		return err
	},
}

func init() {
	configCmd.AddCommand(configAddCmd)

	configAddCmd.Flags().StringP("bucket", "b", "", "Bucket name")
	configAddCmd.Flags().StringP("endpoint", "e", "", "Bucket endpoint")
	configAddCmd.Flags().StringP("region", "r", "", "Bucket region")
	configAddCmd.Flags().StringP("alias", "a", "", "Bucket alias")
	configAddCmd.Flags().BoolP("ofs", "o", false, "Bucket ofs")

	_ = configAddCmd.MarkFlagRequired("bucket")
	// _ = configAddCmd.MarkFlagRequired("endpoint")
}

func addBucketConfig(cmd *cobra.Command) error {
	name, _ := cmd.Flags().GetString("bucket")
	endpoint, _ := cmd.Flags().GetString("endpoint")
	region, _ := cmd.Flags().GetString("region")
	alias, _ := cmd.Flags().GetString("alias")
	ofs, _ := cmd.Flags().GetBool("ofs")

	if alias == "" {
		alias = name
	}
	bucket := util.Bucket{
		Name:     name,
		Endpoint: endpoint,
		Region:   region,
		Alias:    alias,
		Ofs:      ofs,
	}

	for _, b := range config.Buckets {
		if name == b.Name {
			return fmt.Errorf("The bucket already exists, fail to add!")
		} else if alias == b.Name {
			return fmt.Errorf("The alias cannot be the same as other bucket's name")
		} else if alias == b.Alias {
			return fmt.Errorf("The alias already exists, fail to add!")
		}
	}

	config.Buckets = append(config.Buckets, bucket)
	viper.Set("cos.buckets", config.Buckets)

	// 判断config文件是否存在。不存在则创建
	home, err := homedir.Dir()
	configFile := ""
	if cfgFile != "" {
		if cfgFile[0] == '~' {
			configFile = home + cfgFile[1:]
		} else {
			configFile = cfgFile
		}
	} else {
		configFile = home + "/.cos.yaml"
	}
	_, err = os.Stat(configFile)
	if os.IsNotExist(err) || cfgFile != "" {
		if err := viper.WriteConfigAs(configFile); err != nil {
			return err
		}
	} else {
		if err := viper.WriteConfigAs(viper.ConfigFileUsed()); err != nil {
			return err
		}
	}
	logger.Infof("Add successfully! name: %s, endpoint: %s, alias: %s\n, ofs: %t\n", name, endpoint, alias, ofs)
	return nil
}
