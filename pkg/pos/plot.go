package pos

import (
	"os"

	"github.com/spf13/afero"

	fsutil "github.com/kargakis/chiapos/pkg/utils/fs"
)

// PlotDisk is the main function that handles executing all the different
// steps required to plot a disk.
func PlotDisk(filename, fsType string, k, availableMemory int, id []byte, retry bool) (int, error) {
	fs, err := fsutil.GetFs(fsType)
	if err != nil {
		return 0, err
	}

	var file afero.File
	if retry {
		file, err = fs.OpenFile(filename, os.O_RDWR, 0)
	} else {
		file, err = fs.Create(filename)
	}
	if err != nil {
		return 0, err
	}
	defer file.Close()

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
