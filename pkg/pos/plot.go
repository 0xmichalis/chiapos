package pos

import (
	"fmt"
	"math"
	"time"

	"github.com/spf13/afero"

	"github.com/kargakis/gochia/pkg/serialize"
	"github.com/kargakis/gochia/pkg/utils"
	"github.com/kargakis/gochia/pkg/utils/sort"
)

var AppFs = afero.NewOsFs()

// This is Phase 1, or forward propagation. During this phase, all of the 7 tables,
// and f functions, are evaluated. The result is an intermediate plot file, that is
// several times larger than what the final file will be, but that has all of the
// proofs of space in it. First, F1 is computed, which is special since it uses
// AES256, and each encryption provides multiple output values. Then, the rest of the
// f functions are computed, and a sort on disk happens for each table.
func WritePlotFile(filename string, k, availableMemory uint64, memo, id []byte) error {
	file, err := AppFs.Create(filename)
	if err != nil {
		return err
	}

	headerLen, err := WriteHeader(file, k, memo, id)
	if err != nil {
		return err
	}

	fmt.Println("Computing table 1...")
	start := time.Now()
	f1, err := NewF1(k, id)
	if err != nil {
		return err
	}

	var wrote int
	maxNumber := uint64(math.Pow(2, float64(k)))

	// TODO: Try to parallelize and see how it fares CPU-wise
	for x := uint64(0); x < maxNumber; x++ {
		f1x := f1.Calculate(x)
		n, err := serialize.Write(file, int64(headerLen+wrote), x, f1x, int(k))
		if err != nil {
			return err
		}
		wrote += n
	}
	fmt.Printf("Wrote %d entries (size: %s)\n", maxNumber, utils.PrettySize(uint64(wrote)))

	// if we know beforehand there is not enough space
	// to sort in memory, we can prepare the spare file
	var spare afero.File
	if uint64(wrote) > availableMemory {
		spare, err = AppFs.Create(filename + "-spare")
		if err != nil {
			return err
		}
	}

	fmt.Println("Sorting table 1...")

	if err := sort.OnDisk(file, spare, uint64(headerLen), uint64(wrote+headerLen), availableMemory, uint64(wrote)/maxNumber, maxNumber, int(k)); err != nil {
		return err
	}
	fmt.Printf("F1 calculations finished in %v (wrote %s)\n", time.Since(start), utils.PrettySize(uint64(wrote)))

	fmt.Println("Computing table 2...")
	start = time.Now()
	fx, err := NewFx(k, id)
	if err != nil {
		return err
	}

	for x := uint64(0); x < maxNumber; x++ {
		_ = fx
	}

	return nil
}

// WriteHeader writes the plot file header to a file
// 19 bytes  - "Proof of Space Plot" (utf-8)
// 32 bytes  - unique plot id
// 1 byte    - k
// 2 bytes   - memo length
// x bytes   - memo
func WriteHeader(file afero.File, k uint64, memo, id []byte) (int, error) {
	n, err := file.Write([]byte("Proof of Space Plot"))
	if err != nil {
		return n, err
	}

	nmore, err := file.Write(id)
	n += nmore
	if err != nil {
		return n, err
	}

	nmore, err = file.Write([]byte{byte(k)})
	n += nmore
	if err != nil {
		return n, err
	}

	sizeBuf := make([]byte, 2)
	sizeBuf[0] = byte(len(memo))
	nmore, err = file.Write(sizeBuf)
	n += nmore
	if err != nil {
		return n, err
	}

	nmore, err = file.Write(memo)
	return n + nmore, err
}
