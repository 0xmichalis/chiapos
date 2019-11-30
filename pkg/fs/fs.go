package fs

import (
	"io"
	"io/ioutil"
	"os"
)

// NewOSFile creates a new OS file and returns a generic handler to it.
// If the provided filename is empty, then a temporary file is generated
// and its filename is returned.
func NewOSFile(filename string) (handler io.ReadWriteSeeker, name string, err error) {
	var file *os.File
	if filename == "" {
		file, err = ioutil.TempFile("", "plot-")
	} else {
		file, err = os.Create(filename)
	}
	if err == nil {
		handler = file
		name = file.Name()
	}
	return
}
