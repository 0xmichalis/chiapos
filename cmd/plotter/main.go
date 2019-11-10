package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/kargakis/gochia/pkg/pos"
)

var (
	k        = flag.Uint64("k", 33, "Storage parameter")
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

	if err := pos.WritePlotFile(*plotPath, *k, nil, key[:]); err != nil {
		fmt.Printf("cannot write plot: %v", err)
		os.Exit(1)
	}
}
