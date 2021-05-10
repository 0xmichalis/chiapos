package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/mem"
	"github.com/skycoinsynth/chiapos-go/pkg/pos"
	"github.com/skycoinsynth/chiapos-go/pkg/utils"
	fsutil "github.com/skycoinsynth/chiapos-go/pkg/utils/fs"
)

var (
	retry    = flag.Bool("retry", false, "If set to true, try to restore from a pre-existing plot")
	k        = flag.Int("k", 18, "Storage parameter")
	plotPath = flag.String("f", "plot.dat", "Path to the plot")
	fsType   = flag.String("fs", fsutil.OsType, "Filesystem type")
	keyPath  = flag.String("seed", "", "Path to key to be used as a plot seed")
	availMem = flag.Int("m", 5*1024*1024*1024, "Max memory to use when plotting. Defaults to all OS available memory when set to zero.")
)

func retrieveKey(keyPath, plotPath string, retry bool) ([]byte, error) {
	var key []byte
	var err error

	if retry {
		// Try to retrieve key from pre-existing plot
		fmt.Printf("Reading seed from pre-existing plot at %s...\n", plotPath)
		key, err = pos.GetKey(plotPath)
	} else if keyPath == "" {
		// If a key is not provided, generate one in random
		fmt.Println("Generating seed...")
		key = make([]byte, utils.KeyLen)
		_, err = rand.Read(key)
		if err == nil {
			err = ioutil.WriteFile(".seed", key, 0600)
		}
	} else {
		fmt.Printf("Reading seed from %s...\n", keyPath)
		key, err = ioutil.ReadFile(keyPath)
		if err == nil {
			key = utils.NormalizeKey(key)
		}
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
		v, err := mem.VirtualMemory()
		if err != nil {
			fmt.Printf("cannot read system info to get available memory : %v", err)
		}
		*availMem = int(v.Free)
	}

	// TODO: Re-enable when sort on disk is finalized
	// https://github.com/skycoinsynth/chiapos-go/issues/5
	// fmt.Printf("Available memory: %dMB\n", *availMem/(1024*1024))

	// run GC manually to flush unused memory as quickly as possible
	go gc()

	plotStart := time.Now()
	wrote, err := pos.PlotDisk(*plotPath, *fsType, *k, *availMem, key[:], *retry)
	if err != nil {
		fmt.Printf("cannot write plot: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Plotting: OK (Wrote %v in %v)\n", utils.PrettySize(float64(wrote)), time.Since(plotStart))
}
