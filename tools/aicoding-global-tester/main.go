package main

import (
	"os"

	"github.com/JiaxI2/AiCoding/internal/testengine"
)

func main() {
	os.Exit(testengine.Execute(os.Args[1:], os.Stdout, os.Stderr))
}
