package main

import (
	"crypto/rand"
	"crypto/sha256"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/kargakis/chiapos/pkg/pos"
	fsutil "github.com/kargakis/chiapos/pkg/utils/fs"
)

var (
	c        = flag.String("c", "", "Challenge to use for the space proof")
	plotPath = flag.String("f", "plot.dat", "Path to the plot")
	fsType   = flag.String("fs", fsutil.OsType, "Filesystem type")
)

func main() {
	flag.Parse()

	challenge := []byte(*c)
	if len(challenge) == 0 {
		challenge = make([]byte, 32)
		if _, err := rand.Read(challenge); err != nil {
			fmt.Printf("Cannot generate random challenge: %v\n", err)
			os.Exit(1)
		}
		h := sha256.New()
		h.Write(challenge)
		challenge = h.Sum(nil)
		// fmt.Printf("Using random challenge %x\n", challenge)
		if err := ioutil.WriteFile(".random_challenge", challenge, 0600); err != nil {
			fmt.Printf("Cannot persist random challenge: %v\n", err)
			os.Exit(1)
		}
	}
	if len(challenge) != 32 {
		fmt.Printf("Challenge is %d bytes; needs to be 32\n", len(challenge))
		os.Exit(1)
	}

	proof, err := pos.Prove(*plotPath, *fsType, challenge)
	if err != nil {
		fmt.Printf("Cannot read plot: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(proof)
}
