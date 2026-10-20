package main

import (
	"github.com/AwayerCode/Harn/cli/swe-cli/internal/app"
	"os"
)

func main() { os.Exit(app.Run(os.Args[1:], os.Stdin, os.Stdout)) }
