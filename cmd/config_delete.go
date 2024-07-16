package cmd

import (
	"coscli/util"
	"fmt"
	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Used to delete an existing bucket",
	Long: `Used to delete an existing bucket

Format:
  ./coscli config delete -a <alias> [-c <config-file-path>]

Example:
  ./coscli config delete -a example`,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := deleteBucketConfig(cmd)
		return err
	},
}

func init() {
	configCmd.AddCommand(configDeleteCmd)

	configDeleteCmd.Flags().StringP("alias", "a", "", "Bucket alias")

	_ = configDeleteCmd.MarkFlagRequired("alias")
}

func deleteBucketConfig(cmd *cobra.Command) error {
	alias, _ := cmd.Flags().GetString("alias")
	b, i, err := util.FindBucket(&config, alias)
	if err != nil {
		return err
	}

	if i < 0 {
		return fmt.Errorf("Bucket not exist in config file!")
	}
	config.Buckets = append(config.Buckets[:i], config.Buckets[i+1:]...)

	viper.Set("cos.buckets", config.Buckets)
	if err := viper.WriteConfigAs(viper.ConfigFileUsed()); err != nil {
		return err
	}
	logger.Infof("Delete successfully! name: %s, endpoint: %s, alias: %s", b.Name, b.Endpoint, b.Alias)
	return nil
}
