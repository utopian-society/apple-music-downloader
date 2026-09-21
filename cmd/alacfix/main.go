package main

import (
	"flag"
	"fmt"
	"os"

	"amdl/internal/media/alacfix"
)

func main() {
	var inPlace, force, verbose bool
	flag.BoolVar(&inPlace, "i", false, "modify file in place")
	flag.BoolVar(&inPlace, "in-place", false, "modify file in place")
	flag.BoolVar(&force, "f", false, "always write output even if no patches applied")
	flag.BoolVar(&force, "force", false, "always write output even if no patches applied")
	flag.BoolVar(&verbose, "v", false, "verbose output")
	flag.BoolVar(&verbose, "verbose", false, "verbose output")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  alacfix [-f] [-v] <input.m4a> <output.m4a>")
		fmt.Fprintln(os.Stderr, "  alacfix [-f] [-v] -i <input.m4a>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Flags:")
		fmt.Fprintln(os.Stderr, "  -i, --in-place   modify file in place")
		fmt.Fprintln(os.Stderr, "  -f, --force      always write output even if no patches applied")
		fmt.Fprintln(os.Stderr, "  -v, --verbose    verbose output")
	}
	flag.Parse()

	args := flag.Args()

	var input, output string
	switch {
	case inPlace && len(args) == 1:
		input = args[0]
	case !inPlace && len(args) == 2:
		input, output = args[0], args[1]
	default:
		flag.Usage()
		os.Exit(1)
	}

	var err error
	if verbose {
		err = alacfix.RunVerbose(input, force, output)
	} else {
		err = alacfix.Run(input, force, output)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
