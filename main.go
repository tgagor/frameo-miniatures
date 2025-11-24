package main

import (
	"github.com/tgagor/frameo-miniatures/cmd"
)

var BuildVersion string // Will be set dynamically at build time.
var appName string = "frameo-miniatures"

func main() {
	cmd.Execute(appName, BuildVersion)
}
