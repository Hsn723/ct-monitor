package main

import (
	"github.com/Hsn723/ct-monitor/cmd"
	_ "golang.org/x/crypto/x509roots/fallback"
)

func main() {
	cmd.Execute()
}
