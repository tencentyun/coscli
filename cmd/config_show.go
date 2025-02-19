package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Prints information from a specified configuration file",
	Long: `Prints information from a specified configuration file

Format:
  ./coscli config show [-c <config-file-path>]

Example:
  ./coscli config show`,
	Run: func(cmd *cobra.Command, args []string) {
		showConfig()
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
}

func showConfig() {
	fmt.Println("Configuration file path:")
	fmt.Printf("  %s\n", viper.ConfigFileUsed())
	fmt.Println("====================")
	fmt.Println("Basic Configuration Information:")
	fmt.Printf("  Secret ID:     %s\n", config.Base.SecretID)
	fmt.Printf("  Secret Key:    %s\n", config.Base.SecretKey)
	fmt.Printf("  Session Token: %s\n", config.Base.SessionToken)
	fmt.Printf("  Mode: %s\n", config.Base.Mode)
	fmt.Printf("  CvmRoleName: %s\n", config.Base.CvmRoleName)
	fmt.Printf("  CloseAutoSwitchHost: %s\n", config.Base.CloseAutoSwitchHost)
	fmt.Printf("  DisableEncryption: %s\n", config.Base.DisableEncryption)
	fmt.Println("====================")
	fmt.Println("Bucket Configuration Information:")

	for i, b := range config.Buckets {
		fmt.Printf("- Bucket %d :\n", i+1)
		fmt.Printf("  Name:  \t%s\n", b.Name)
		fmt.Printf("  Endpoint:\t%s\n", b.Endpoint)
		fmt.Printf("  Alias: \t%s\n", b.Alias)
		fmt.Printf("  Ofs: \t%v\n", b.Ofs)
	}
}
