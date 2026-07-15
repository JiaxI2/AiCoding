package main

import (
	"fmt"
	"os"

	"c-userstyle-kit/internal/cuserstyle"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: cstylekit <init|demo|snippet|lint|contract|doctor|bench|verify> [args]")
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "init":
		err = cuserstyle.RunInit(os.Args[2:])
	case "demo":
		err = cuserstyle.RunDemo(os.Args[2:])
	case "snippet":
		err = cuserstyle.RunSnippet(os.Args[2:])
	case "lint", "check":
		err = cuserstyle.RunLint(os.Args[2:])
	case "contract":
		err = cuserstyle.RunContract(os.Args[2:])
	case "doctor":
		err = cuserstyle.RunDoctor(os.Args[2:])
	case "bench":
		err = cuserstyle.RunBench(os.Args[2:])
	case "verify":
		err = cuserstyle.RunVerify(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
