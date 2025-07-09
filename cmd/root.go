package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/chatmcp/mcprouter/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "mcprouter",
	Short: "mcprouter is a proxy for mcp server",
	Long: `mcprouter is a proxy for mcp server.

It will forward the request to the mcp server and return the response to the client.
`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func Init() error {
	if err := util.InitConfigWithFile(proxyConfigFile); err != nil {
		fmt.Printf("init config failed with file: %s, %v\n", proxyConfigFile, err)
		return err
	}

	log.Println("config initialized")

	dbNames := []string{viper.GetString("app.db_name")}
	if viper.GetBool("app.use_db") {
		for _, name := range dbNames {
			if name != "" {
				if err := util.InitDBWithName(name); err != nil {
					fmt.Printf("init db failed with name: %s, %v\n", name, err)
					return err
				}
				log.Printf("db %s initialized", name)
			}
		}
	}

	cacheNames := []string{viper.GetString("app.cache_name")}
	if viper.GetBool("app.use_cache") {
		for _, name := range cacheNames {
			if name != "" {
				if err := util.InitRedisWithName(name); err != nil {
					fmt.Printf("init redis failed with name: %s, %v\n", name, err)
					return err
				}
				log.Printf("redis %s initialized", name)
			}
		}
	}

	return nil
}

func init() {
	rootCmd.PersistentFlags().StringP("version", "v", "0.0.1", "version")
}
