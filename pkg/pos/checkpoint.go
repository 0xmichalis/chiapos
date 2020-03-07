package pos

import "github.com/spf13/afero"

// Checkpoint reads the last table in the plot and creates a new
// table where it stores checkpoints to the last table so fast
// retrieval of proofs can be enabled by reading the checkpoints.
func Checkpoint(file afero.File, k int) (int, error) {
	return 0, nil
}
