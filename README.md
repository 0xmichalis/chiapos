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

Plot your disk with:
```
./bin/plotter
```

TODO: Proofs and verification

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
