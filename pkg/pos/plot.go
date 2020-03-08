package pos

import (
	"os"

	"github.com/spf13/afero"
)

// PlotDisk is the main function that handles executing all the different
// steps required to plot a disk.
func PlotDisk(filename string, k, availableMemory int, id []byte, retry bool) (int, error) {
	fs := afero.NewOsFs()

	var file afero.File
	var err error
	if retry {
		file, err = fs.OpenFile(filename, os.O_RDWR, 0)
	} else {
		file, err = fs.Create(filename)
	}
	if err != nil {
		return 0, err
	}

	// Run forward propagation
	wrote, err := ForwardPropagate(fs, file, k, availableMemory, id, retry)
	if err != nil {
		return wrote, err
	}

	// Checkpoint the last table so we can retrieve proofs as
	// fast as possible.
	cWrote, err := Checkpoint(file, k)
	return cWrote + wrote, err
}
