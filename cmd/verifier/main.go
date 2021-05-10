package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/skycoinsynth/chiapos-go/pkg/pos"
	"github.com/skycoinsynth/chiapos-go/pkg/utils"
)

var (
	c       = flag.String("c", "", "Challenge to use for the space proof")
	k       = flag.Int("k", 18, "Space parameter")
	keyPath = flag.String("seed", "", "Path to the plot seed")
	proof   = flag.String("p", "", "Space proof")
)

func main() {
	flag.Parse()

	seed, err := ioutil.ReadFile(*keyPath)
	if err != nil {
		fmt.Printf("Cannot set up plot seed: %v\n", err)
		os.Exit(1)
	}
	seed = utils.NormalizeKey(seed)

	if *c == "" {
		fmt.Println("Challenge cannot be empty")
		os.Exit(1)
	}

	if *proof == "" {
		fmt.Println("Space proof cannot be empty")
		os.Exit(1)
	}

	proofStrings := strings.Split(*proof, ",")
	if len(proofStrings) != 64 {
		fmt.Printf("Invalid space proof: expected 64 values, got %d\n", len(proofStrings))
		os.Exit(1)
	}

	var proofs []uint64
	for _, p := range proofStrings {
		pi, err := strconv.Atoi(strings.TrimRight(p, ","))
		if err != nil {
			fmt.Printf("Invalid space proof (%s): %v\n", *proof, err)
			os.Exit(1)
		}
		proofs = append(proofs, uint64(pi))
	}

	if err := pos.Verify(*c, seed, *k, proofs); err != nil {
		fmt.Printf("Cannot verify space proof: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("The provided space proof is valid.")
}
