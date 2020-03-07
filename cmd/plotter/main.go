package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"syscall"
	"time"

	"github.com/kargakis/gochia/pkg/pos"
	"github.com/kargakis/gochia/pkg/utils"
)

var (
	retry    = flag.Bool("retry", false, "If set to true, try to restore from a pre-existing plot")
	k        = flag.Int("k", 15, "Storage parameter")
	plotPath = flag.String("f", "plot.dat", "Final path to the plot")
	keyPath  = flag.String("key", "", "Path to key to be used as a plot seed")
	availMem = flag.Int("m", 5*1024*1024*1024, "Max memory to use when plotting. Defaults to all OS available memory when set to zero.")
)

func retrieveKey(keyPath, plotPath string, retry bool) ([32]byte, error) {
	var key [32]byte
	var err error

	if retry {
		// Try to retrieve key from pre-existing plot
		fmt.Printf("Reading seed from pre-existing plot at %s...\n", plotPath)
		key, err = pos.GetKey(plotPath)
	} else if keyPath == "" {
		// If a key is not provided, generate one in random
		fmt.Println("Generating seed...")
		_, err = rand.Read(key[:])
	} else {
		fmt.Printf("Reading seed from %s...\n", keyPath)
		_, err = ioutil.ReadFile(keyPath)
	}

	return key, err
}

func gc() {
	for {
		select {
		case <-time.Tick(10 * time.Second):
			// TODO: Run this only we actually need to manually
			// free up memory instead of every 10 seconds.
			runtime.GC()
		}
	}
}

func main() {
	flag.Parse()

	key, err := retrieveKey(*keyPath, *plotPath, *retry)
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

	// run GC manually to flush unused memory as quickly as possible
	go gc()

	fmt.Printf("Generating plot at %s with k=%d\n", *plotPath, *k)
	plotStart := time.Now()
	wrote, err := pos.PlotDisk(*plotPath, *k, *availMem, key[:], *retry)
	if err != nil {
		fmt.Printf("cannot write plot: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Plotting: OK (Wrote %v in %v)\n", utils.PrettySize(wrote), time.Since(plotStart))
}
