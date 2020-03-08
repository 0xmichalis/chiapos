package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/kargakis/chiapos/pkg/pos"
)

var (
	c        = flag.String("c", "", "Challenge to use for the space proof")
	plotPath = flag.String("f", "plot.dat", "Final path to the plot")
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
		fmt.Printf("Using random challenge %s\n", challenge)
		if err := ioutil.WriteFile(".random_challenge", challenge, 0777); err != nil {
			fmt.Printf("Cannot persist random challenge: %v\n", err)
			os.Exit(1)
		}
	}
	if len(challenge) != 32 {
		fmt.Println("Challenge needs to be 256 bits")
		os.Exit(1)
	}

	proof, err := pos.Prove(*plotPath, challenge)
	if err != nil {
		fmt.Printf("Cannot read plot: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(proof)
}
