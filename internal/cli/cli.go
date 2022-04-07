package cli

import (
	"fmt"
	"os"

	"github.com/grussorusso/serverledge/internal/config"
	"github.com/spf13/cobra"
)

var ServerConfig config.RemoteServerConf

var rootCmd = &cobra.Command{
	Use:   "serverledge-cli",
	Short: "CLI utility for Serverledge",
	Long:  `CLI utility to interact with a Serverledge FaaS platform.`,
}

var invokeCmd = &cobra.Command{
	Use:   "invoke",
	Short: "Invokes a function",
	Run:   invoke,
}

var funcName, runtime, handler, customImage, src, qosClass string
var memory int64
var cpuDemand, qosMaxRespT float64
var params []string
var verbose bool

func Init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVarP(&ServerConfig.Host, "host", "H", ServerConfig.Host, "remote Serverledge host")
	rootCmd.PersistentFlags().IntVarP(&ServerConfig.Port, "port", "P", ServerConfig.Port, "remote Serverledge port")

	rootCmd.AddCommand(invokeCmd)
	invokeCmd.Flags().StringVarP(&funcName, "function", "f", "", "name of the function")
	invokeCmd.Flags().Float64VarP(&qosMaxRespT, "maxRespT", "", -1.0, "Max. response time (optional)")
	invokeCmd.Flags().StringVarP(&qosClass, "qosClass", "c", "", "QoS class (optional)")
	invokeCmd.Flags().StringSliceVarP(&params, "param", "p", nil, "Function parameter: <name>:<value>")

	//TODO
	//        //parse parameters for function creation
	//        flag.StringVar(&runtime, "runtime", "python38", "runtime for the function")
	//        flag.StringVar(&handler, "handler", "", "function handler")
	//        flag.StringVar(&customImage, "custom_image", "", "custom container image (only if runtime is 'custom')")
	//        flag.Int64Var(&memory, "memory", 128, "max memory in MB for the function")
	//        flag.Float64Var(&cpuDemand, "cpu", 0.0, "estimated CPU demand for the function (e.g., 1.0 = 1 core)")
	//        flag.StringVar(&src, "src", "", "source the function (single file, directory or TAR archive)")

	//        case command == "create":
	//                cli.Create(funcName, runtime, customImage, src, handler, memory, cpuDemand)
	//        case command == "delete":
	//                cli.Delete()
	//        case command == "list":
	//                cli.List()
	//        case command == "status":
	//                cli.GetStatus()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
