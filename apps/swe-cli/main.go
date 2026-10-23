package main

import (
	"github.com/AwayerCode/Harn/apps/swe-cli/internal/app"
	"os"
)

func main() { os.Exit(app.RunInstalled(os.Args[1:], os.Stdin, os.Stdout)) }
