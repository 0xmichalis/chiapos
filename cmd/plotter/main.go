package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/kargakis/gochia/pkg/pos"
)

var (
	k        = flag.Uint64("k", 15, "Storage parameter")
	plotPath = flag.String("f", "", "Final path to the plot")
	keyPath  = flag.String("key", "", "Path to key to be used as a plot seed")
)

func main() {
	flag.Parse()

	// If a key is not provided, generate one in random
	var key [32]byte
	var err error
	if *keyPath == "" {
		fmt.Println("Generating private key...")
		_, err = rand.Read(key[:])
	} else {
		fmt.Printf("Reading private key from %s...\n", *keyPath)
		_, err = ioutil.ReadFile(*keyPath)
	}
	if err != nil {
		fmt.Printf("cannot set up plot seed: %v", err)
		os.Exit(1)
	}

	// If a plot path is not provided, use a temporary file
	var plot string
	if *plotPath != "" {
		plot = *plotPath
	} else {
		plotFile, err := ioutil.TempFile("", "plot-")
		if err != nil {
			fmt.Printf("cannot set up plot file: %v", err)
			os.Exit(1)
		}
		plot = plotFile.Name()
	}

	fmt.Printf("Generating plot at %s\n", plot)
	plotStart := time.Now()
	if err := pos.WritePlotFile(plot, *k, nil, key[:]); err != nil {
		fmt.Printf("cannot write plot: %v", err)
		os.Exit(1)
	}
	fmt.Printf("Plotting: OK (%v)\n", time.Since(plotStart))
}
