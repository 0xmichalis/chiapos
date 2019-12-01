package pos

import (
	"fmt"
	"math"
	"time"

	"github.com/spf13/afero"

	"github.com/kargakis/gochia/pkg/parameters"
	"github.com/kargakis/gochia/pkg/utils/sort"
)

const (
	// EPP for the final file, the higher this is, the less variability, and lower delta
	// Note: if this is increased, ParkVector size must increase
	kEntriesPerPark = 2048

	// To store deltas for EPP entries, the average delta must be less than this number of bits
	kMaxAverageDeltaTable1 = 5.6
	kMaxAverageDelta       = 3.5

	// C3 entries contain deltas for f7 values, the max average size is the following
	kC3BitsPerEntry = 2.4

	// The number of bits in the stub is k minus this value
	kStubMinusBits = 3

	// Constants that are only relevant for the plotting process.
	// Other constants can be found in pos_constants.hpp
	kMemorySize = 2147483648 // 2^31, or 2GB

	// Number of buckets to use for SortOnDisk.
	kNumSortBuckets = 16

	// During back propagation and compress, the write pointer is ahead of the read pointer
	// Note that the large the offset, the higher these values must be
	kReadMinusWrite      = 2048
	kCachedPositionsSize = 8192

	// Distance between matching entries is stored in the offset
	kOffsetSize = 11

	// Max matches a single entry can have, used for hardcoded memory allocation
	kMaxMatchesSingleEntry = 30

	// Unique plot id which will be used as an AES key, and determines the PoSpace.
	kIdLen = 32

	// Must be set high enough to prevent attacks of fast plotting
	// TODO: Should be set to 33
	kMinPlotSize = 15

	// Set at 59 to allow easy use of 64 bit integers
	kMaxPlotSize = 59

	kFormatDescription = "alpha-v0.4"
)

func CreatePlotDisk(filename string, k uint64, memo, id []byte) error {
	fmt.Printf("Starting plotting progress into file %s\n", filename)

	// These variables are used in the WriteParkToFile method. They are pre-allocated here
	// to save time.
	// first_line_point_bytes := CalculateLinePointSize(k)
	// park_stubs_bytes := CalculateStubsSize(k)
	// park_deltas_bytes := CalculateMaxDeltasSize(k, 1)

	if len(id) != kIdLen {
		return fmt.Errorf("invalid id length: %d", len(id))
	}
	if k < kMinPlotSize || k > kMaxPlotSize {
		return fmt.Errorf("invalid k size: %d", k)
	}

	return nil
}

func CalculateLinePointSize(k int) int {
	return byteAlign(2 * k)
}

func byteAlign(numBits int) int {
	return (numBits + (8-(numBits%8))%8) / 8
}

func CalculateStubsSize(k int) int {
	return byteAlign((kEntriesPerPark - 1) * (k - kStubMinusBits))
}

// This is the full size of the deltas section in a park. However, it will not be fully filled
func CalculateMaxDeltasSize(k, tableIndex int) int {
	var bits float64
	if tableIndex == 1 {
		bits = (kEntriesPerPark - 1) * kMaxAverageDeltaTable1
	} else {
		bits = (kEntriesPerPark - 1) * kMaxAverageDelta
	}
	return byteAlign(int(bits))
}

// This is Phase 1, or forward propagation. During this phase, all of the 7 tables,
// and f functions, are evaluated. The result is an intermediate plot file, that is
// several times larger than what the final file will be, but that has all of the
// proofs of space in it. First, F1 is computed, which is special since it uses
// AES256, and each encryption provides multiple output values. Then, the rest of the
// f functions are computed, and a sort on disk happens for each table.
func WritePlotFile(file afero.File, k, availableMemory uint64, memo, id []byte) error {
	headerLen, err := WriteHeader(file, k, memo, id)
	if err != nil {
		return err
	}

	maxNumber := uint64(math.Pow(2, float64(k)))
	maxDigits := countDigits(maxNumber - 1)
	maxEncryptedDigits := countDigits(uint64(math.Pow(2, float64(k+parameters.ParamEXT))) - 1)

	fmt.Println("Computing table 1...")
	start := time.Now()
	f1, err := NewF1(k, id)
	if err != nil {
		return err
	}

	var wrote int
	// TODO: Try to parallelize and see how it fares CPU-wise
	for x := uint64(0); x < maxNumber; x++ {
		f1x := f1.Calculate(x)
		// TODO: Batch writes
		fullyPrint := fmt.Sprintf("%%0%dx,%%0%dx\n", maxEncryptedDigits, maxDigits)
		n, err := file.Write([]byte(fmt.Sprintf(fullyPrint, f1x, x)))
		if err != nil {
			return err
		}
		wrote += n
	}

	if err := sort.OnDisk(file, uint64(headerLen), uint64(wrote+headerLen), availableMemory, uint64(wrote)/maxNumber); err != nil {
		return err
	}
	fmt.Printf("F1 calculations finished in %v (wrote %s)\n", time.Since(start), prettySize(uint64(wrote)))

	return nil
}

func countDigits(number uint64) int {
	if number < 16 {
		return 1
	}
	return 1 + countDigits(number/16)
}

func prettySize(size uint64) string {
	switch c := class(size); c {
	case 0: // bytes
		return fmt.Sprintf("%d B", size)
	case 1: // kilobytes
		return fmt.Sprintf("%d KB", size/1024)
	case 2: // megabytes
		return fmt.Sprintf("%d MB", size/(1024*1024))
	default: // gigabytes
		return fmt.Sprintf("%d GB", size/(1024*1024*1024))
	}
}

func class(size uint64) int {
	if size < 1024 {
		return 0
	}
	return 1 + class(size/1024)
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
