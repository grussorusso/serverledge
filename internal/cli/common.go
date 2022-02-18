package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/grussorusso/serverledge/internal/config"
)

var ServerConfig config.RemoteServerConf

func ExitWithUsage() {
	fmt.Println("Expected a subcommand: invoke|create|delete|list|status")
	fmt.Println("Get help for a specific subcommand appending '-h'")
	flag.Usage()
	os.Exit(1)
}
