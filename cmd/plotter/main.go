package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"syscall"
	"time"

	"github.com/kargakis/gochia/pkg/pos"
)

var (
	k        = flag.Int("k", 15, "Storage parameter")
	plotPath = flag.String("f", "plot.dat", "Final path to the plot")
	keyPath  = flag.String("key", "", "Path to key to be used as a plot seed")
	availMem = flag.Int("m", 5*1024*1024*1024, "Max memory to use when plotting. Defaults to all OS available memory when set to zero.")
)

func main() {
	flag.Parse()

	// If a key is not provided, generate one in random
	var key [32]byte
	var err error
	if *keyPath == "" {
		fmt.Println("Generating seed...")
		_, err = rand.Read(key[:])
	} else {
		fmt.Printf("Reading seed from %s...\n", *keyPath)
		_, err = ioutil.ReadFile(*keyPath)
	}
	if err != nil {
		fmt.Printf("cannot set up plot seed: %v", err)
		os.Exit(1)
	}

	if *availMem == 0 {
		si := &syscall.Sysinfo_t{}
		if err := syscall.Sysinfo(si); err != nil {
			fmt.Printf("cannot read system info to get available memory: %v", err)
			os.Exit(1)
		}
		*availMem = int(si.Freeram)
	}
	fmt.Printf("Available memory: %dMB\n", *availMem/(1024*1024))

	fmt.Printf("Generating plot at %s\n", *plotPath)
	plotStart := time.Now()
	if err := pos.WritePlotFile(*plotPath, *k, *availMem, nil, key[:]); err != nil {
		fmt.Printf("cannot write plot: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Plotting: OK (%v)\n", time.Since(plotStart))
}
