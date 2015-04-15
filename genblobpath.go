package otaru

import (
	"encoding/hex"
	"fmt"
)

func GenerateNewBlobPath(bs RandomAccessBlobStore) (string, error) {
	const MaxTrial = 256
	const BlobPathLen = 16

	for i := 0; i < MaxTrial; i++ {
		randbin := RandomBytes(BlobPathLen)
		candidate := hex.EncodeToString(randbin)

		bh, err := bs.Open(candidate)
		if err != nil {
			return "", err
		}
		if bh.Size() == 0 {
			return candidate, nil
		}
		if err := bh.Close(); err != nil {
			return "", err
		}
	}
	return "", fmt.Errorf("Failed to generate unique blobpath within %d trials", MaxTrial)
}
