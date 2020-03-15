package fs

import (
	"fmt"

	"github.com/spf13/afero"
)

const (
	OsType = "os"
)

var supportedTypes = []string{OsType}

func GetFs(fs string) (afero.Fs, error) {
	switch fs {
	case OsType:
		return afero.NewOsFs(), nil
	}
	return nil, fmt.Errorf("unknown filesystem type provided: %s (supported types: %v)", fs, supportedTypes)
}
