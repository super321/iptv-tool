package cmds

import (
	"iptv/internal/pkg/util"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

func init() {
	cobra.OnInitialize(initConfig)
}

func NewRootCLI() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "iptv",
		Short:         "IPTV工具",
		SilenceUsage:  true,
		SilenceErrors: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}

	rootCmd.AddCommand(NewKeyCLI())
	rootCmd.AddCommand(NewChannelCLI())
	rootCmd.AddCommand(NewServeCLI())
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "JSON配置文件的路径")

	return rootCmd
}

// initConfig 初始化配置文件
func initConfig() {
	if cfgFile != "" {
		// 使用命令参数中的配置文件
		viper.SetConfigFile(cfgFile)
	} else {
		cfgHome, err := util.GetCurrentAbPathByExecutable()
		cobra.CheckErr(err)

		viper.AddConfigPath(cfgHome)
		viper.SetConfigName("config")
		viper.SetConfigType("json")

		// 创建配置目录
		if _, err = os.Stat(cfgHome); os.IsNotExist(err) {
			err = os.MkdirAll(cfgHome, 0755)
			cobra.CheckErr(err)
		}
		_ = viper.SafeWriteConfig()
	}

	// 读取环境变量
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	cobra.CheckErr(err)
}
