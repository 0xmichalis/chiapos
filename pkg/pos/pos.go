package pos

import (
	"fmt"
	"os"
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

	// During backprop and compress, the write pointer is ahead of the read pointer
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
	kMinPlotSize = 15

	// Set at 59 to allow easy use of 64 bit integers
	kMaxPlotSize = 59

	kFormatDescription = "alpha-v0.4"
)

func CreatePlotDisk(filename string, k int, memo, id []byte) error {
	fmt.Printf("Starting plotting progress into file %s\n", filename)

	// These variables are used in the WriteParkToFile method. They are preallocatted here
	// to save time.
	first_line_point_bytes := CalculateLinePointSize(k)
	park_stubs_bytes := CalculateStubsSize(k)
	park_deltas_bytes := CalculateMaxDeltasSize(k, 1)

	if len(id) != kIdLen {
		return fmt.Errorf("invalid id length: %d", len(id))
	}
	if k < kMinPlotSize || k > kMaxPlotSize {
		return fmt.Errorf("invalid k size: %d", k)
	}

	return nil
}

func CalculateLinePointSize(k int) int {
	return byteAlign(2*k) / 8
}

func byteAlign(numBits int) int {
	return (numBits + (8-(numBits%8))%8)
}

func CalculateStubsSize(k int) int {
	return byteAlign((kEntriesPerPark-1)*(k-kStubMinusBits)) / 8
}

// This is the full size of the deltas section in a park. However, it will not be fully filled
func CalculateMaxDeltasSize(k, table_index int) int {
	if table_index == 1 {
		return byteAlign((kEntriesPerPark-1)*kMaxAverageDeltaTable1) / 8
	}
	return byteAlign((kEntriesPerPark-1)*kMaxAverageDelta) / 8
}

// WriteHeader writes the plot file header to a file
// 19 bytes  - "Proof of Space Plot" (utf-8)
// 32 bytes  - unique plot id
// 1 byte    - k
// 2 bytes   - format description length
// x bytes   - format description
// 2 bytes   - memo length
// x bytes   - memo
func WriteHeader(file *os.File, k int, memo, id []byte) error {
	if _, err := file.Write([]byte("Proof of Space Plot")); err != nil {
		return err
	}
	if _, err := file.Write(id); err != nil {
		return err
	}
	if _, err := file.Write([]byte{byte(k)}); err != nil {
		return err
	}
	sizeBuf := make([]byte, 2)
	sizeBuf[0] = byte(len(kFormatDescription))
	if _, err := file.Write(sizeBuf); err != nil {
		return err
	}
	if _, err := file.Write([]byte(kFormatDescription)); err != nil {
		return err
	}
	sizeBuf[0] = byte(len(memo))
	if _, err := file.Write(sizeBuf); err != nil {
		return err
	}
	if _, err := file.Write(memo); err != nil {
		return err
	}
	return nil
}

