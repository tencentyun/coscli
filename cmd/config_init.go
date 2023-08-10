package cmd

import (
	"coscli/util"
	"fmt"
	"os"

	"github.com/spf13/viper"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Used to interactively generate the configuration file",
	Long: `Used to interactively generate the configuration file

Format:
  ./coscli config init [-c <config-file-path>]

Example:
  ./coscli config init`,
	Run: func(cmd *cobra.Command, args []string) {
		if cmdCnt >= 1 {
			return
		}
		initConfigFile(true)
	},
}

func init() {
	configCmd.AddCommand(configInitCmd)
}

// cfgFlag: 是否允许用户自定义配置文件的输出路径
func initConfigFile(cfgFlag bool) {
	var (
		configFile string
		config     util.Config
		bucket     util.Bucket
	)
	home, _ := homedir.Dir()

	if cfgFlag {
		fmt.Println("Specify the path of the configuration file: (default:" + home + "/.cos.yaml)")
		_, _ = fmt.Scanf("%s\n", &configFile)
	}
	if configFile == "" {
		configFile = home + "/.cos.yaml"
	}
	if configFile[0] == '~' {
		configFile = home + configFile[1:]
	}
	fmt.Println("The path of the configuration file: " + configFile)

	fmt.Println("Input Your Secret ID:")
	_, _ = fmt.Scanf("%s\n", &config.Base.SecretID)
	fmt.Println("Input Your Secret Key:")
	_, _ = fmt.Scanf("%s\n", &config.Base.SecretKey)
	fmt.Println("Input Your Session Token:")
	_, _ = fmt.Scanf("%s\n", &config.Base.SessionToken)
	fmt.Println("Input Your Mode:")
	_, _ = fmt.Scanf("%s\n", &config.Base.Mode)
	fmt.Println("Input Your Cvm Role Name:")
	_, _ = fmt.Scanf("%s\n", &config.Base.CvmRoleName)
	if len(config.Base.SessionToken) < 3 {
		config.Base.SessionToken = ""
	}
	config.Base.Protocol = "https"

	fmt.Println("Input Your Bucket's Name:")
	fmt.Println("Format: <bucketname>-<appid>，Example: example-1234567890")
	_, _ = fmt.Scanf("%s\n", &bucket.Name)
	fmt.Println("Input Bucket's Endpoint:")
	fmt.Println("Format: cos.<region>.myqcloud.com，Example: cos.ap-beijing.myqcloud.com")
	_, _ = fmt.Scanf("%s\n", &bucket.Endpoint)
	fmt.Println("Input Bucket's Alias: (Input nothing will use the original name)")
	_, _ = fmt.Scanf("%s\n", &bucket.Alias)
	if bucket.Alias == "" {
		bucket.Alias = bucket.Name
	}

	config.Buckets = append(config.Buckets, bucket)
	fmt.Println("You have configured the bucket:")
	for _, b := range config.Buckets {
		fmt.Printf("- Name: %s\tEndpoint: %s\tAlias: %s\n", b.Name, b.Endpoint, b.Alias)
	}
	fmt.Printf("\nIf you want to configure more buckets, you can use the \"config add\" command later.\n")
	// 默认加密存储
	config.Base.SecretKey, _ = util.EncryptSecret(config.Base.SecretKey)
	config.Base.SecretID, _ = util.EncryptSecret(config.Base.SecretID)
	config.Base.SessionToken, _ = util.EncryptSecret(config.Base.SessionToken)

	viper.Set("cos", config)

	if err := viper.WriteConfigAs(configFile); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("\nThe configuration file is initialized successfully! \nYou can use \"./coscli config show [-c <Config File Path>]\" show the contents of the specified configuration file\n")
}
