package cli

import (
	"fmt"
	"os"

	"github.com/grussorusso/serverledge/internal/config"
	"github.com/spf13/cobra"
)

var ServerConfig config.RemoteServerConf

var RootCmd = &cobra.Command{
	Use:   "serverledge-cli",
	Short: "CLI utility for Serverledge",
	Long:  `CLI utility to interact with a Serverledge FaaS platform.`,
}

var InvokeCmd = &cobra.Command{
	Use:   "invoke",
	Short: "Invokes a function",
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
		fmt.Printf("Hello!")
	},
}

var Verbose bool

func Init() {
	RootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose output")
	RootCmd.PersistentFlags().StringVarP(&ServerConfig.Host, "host", "H", ServerConfig.Host, "remote Serverledge host")
	RootCmd.PersistentFlags().IntVarP(&ServerConfig.Port, "port", "P", ServerConfig.Port, "remote Serverledge port")

	RootCmd.AddCommand(InvokeCmd)

	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
