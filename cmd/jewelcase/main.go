package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/csmith/jewelcase"
)

func main() {
	var (
		colourCorrection = flag.Bool("colour", true, "Apply colour correction effect")
		roundedCorners   = flag.Bool("corners", true, "Apply rounded corners effect")
		edgeSoftening    = flag.Bool("edges", true, "Apply edge softening effect")
		randomOffset     = flag.Bool("offset", true, "Apply random position offset")
		randomRotation   = flag.Bool("rotation", true, "Apply random rotation")
		reflection       = flag.Bool("reflection", true, "Apply reflection effect")
		inplace          = flag.Bool("inplace", false, "Modify file in-place")
		recursive        = flag.Bool("recursive", false, "Process directory recursively")
		force            = flag.Bool("force", false, "Process images even if they appear to be already processed")
	)
	flag.Parse()

	args := flag.Args()

	opts := jewelcase.Options{
		ColourCorrection: *colourCorrection,
		RoundedCorners:   *roundedCorners,
		EdgeSoftening:    *edgeSoftening,
		RandomOffset:     *randomOffset,
		RandomRotation:   *randomRotation,
		Reflection:       *reflection,
		Force:            *force,
	}

	if *recursive {
		if len(args) != 1 {
			printUsage()
		}
		processDirectory(args[0], opts)
	} else if *inplace {
		if len(args) != 1 {
			printUsage()
		}
		err := jewelcase.ProcessFile(args[0], args[0], opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error applying jewel case: %v\n", err)
			os.Exit(1)
		}
	} else {
		if len(args) != 2 {
			printUsage()
		}
		err := jewelcase.ProcessFile(args[0], args[1], opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error applying jewel case: %v\n", err)
			os.Exit(1)
		}
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options] --recursive <directory>\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "   or: %s [options] --inplace <image>\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "   or: %s [options] <input-image> <output-image>\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Options:\n")
	flag.PrintDefaults()
	os.Exit(1)
}

func processDirectory(dir string, opts jewelcase.Options) {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
			err := jewelcase.ProcessFile(path, path, opts)
			if err != nil {
				if errors.Is(err, jewelcase.ErrAlreadyProcessed) {
					fmt.Printf("Skipped: %s (already processed)\n", path)
				} else {
					fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", path, err)
				}
			} else {
				fmt.Printf("Processed: %s\n", path)
			}
		}

		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking directory: %v\n", err)
		os.Exit(1)
	}
}
