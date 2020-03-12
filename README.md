# Proof of Space

This is an alternative implementation of the [Chia](https://www.chia.net/) Proof of Space algorithm. It was carried out
as part of my Master thesis and is far from a complete implementation so if you are looking for running a Chia farmer 
you should use the [reference implementation](https://github.com/Chia-Network/chia-blockchain).

My intention is to work towards making this a viable alternative to the reference implementation so if you are into
Golang and think Proofs of Space sounds cool, feel free to contribute!

## Build

[Golang](https://golang.org/) is the only requirement to build this project.
Once you have it installed:
```
make build
```

## Run

Create a seed at `.seed` and plot your disk with:
```
./bin/plotter -key .seed
```
The seed is optional and if none is provided, the plotter will generate its own but using our own is convenient in order
to use it for verification below.

Now, search for a proof. We can provide a challenge via the `-c` flag. If no challenge is provided, a random challenge
is generated and persisted at `.random_challenge`. It may happen that we will not find a proof of space immediately
because none exists for the provided challenge. If so, try with a different challenge until one is found.
```
./bin/prover
```
Once we find a proof, we can reproduce the proof retrieval by using the persisted random challenge:
```
./bin/prover -c $(cat .random_challenge) > .proof
```

Now that we have also persisted the proof, we can verify it:
```
./bin/verifier -key .seed -p $(cat .proof) -c $(cat .random_challenge)
```

## Contribute

### Run tests

```
make test
```

### Run code verification

```
make verify
```

### Run benchmarks

```
make bench
```
