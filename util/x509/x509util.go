package x509util

import (
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"fmt"
)

var commonName = asn1.ObjectIdentifier{2, 5, 4, 3}

func FindServerName(rawsubjs [][]byte) (string, error) {
	for _, rawsubj := range rawsubjs {
		var rdns pkix.RDNSequence
		_, err := asn1.Unmarshal(rawsubj, &rdns)
		if err != nil {
			return "", fmt.Errorf("failed to parse subj asn1: %v", err)
		}

		var name pkix.Name
		name.FillFromRDNSequence(&rdns)
		for _, n := range name.Names {
			// logger.Debugf(mylog, "%d %v name: %v", i, n.Type, n.Value)
			if n.Type.Equal(commonName) {
				return n.Value.(string), nil
			}
		}
	}

	return "", errors.New("Couldn't find any commonName")
}
