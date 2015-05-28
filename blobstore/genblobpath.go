package blobstore

import (
	"encoding/hex"
	"fmt"

	"github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/util"
)

// GenerateNewBlobPath tries to return a new unique blob path.
// Note that this may return an already used blobpath in high contention, although it is highly unlikely it will happen.
func GenerateNewBlobPath(bs RandomAccessBlobStore) (string, error) {
	const MaxTrial = 256
	const BlobPathLen = 16

	for i := 0; i < MaxTrial; i++ {
		randbin := util.RandomBytes(BlobPathLen)
		candidate := hex.EncodeToString(randbin)

		bh, err := bs.Open(candidate, flags.O_RDONLY)
		if err != nil {
			if err == ENOENT {
				return candidate, nil
			}
			return "", err
		}
		seemsNotUsed := bh.Size() == 0
		if err := bh.Close(); err != nil {
			return "", err
		}

		if seemsNotUsed {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("Failed to generate unique blobpath within %d trials", MaxTrial)
}
