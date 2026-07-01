package protocol

import (
	"strings"

	"github.com/coreos/go-semver/semver"
	libp2pprotocol "github.com/libp2p/go-libp2p/core/protocol"
)

func multistreamSemverMatcher(base libp2pprotocol.ID) (func(libp2pprotocol.ID) bool, error) {
	parts := strings.Split(string(base), "/")
	vers, err := semver.NewVersion(parts[len(parts)-1])
	if err != nil {
		return nil, err
	}

	return func(check libp2pprotocol.ID) bool {
		chparts := strings.Split(string(check), "/")
		if len(chparts) != len(parts) {
			return false
		}

		for i, v := range chparts[:len(chparts)-1] {
			if parts[i] != v {
				return false
			}
		}

		chvers, err := semver.NewVersion(chparts[len(chparts)-1])
		if err != nil {
			return false
		}

		return vers.Major == chvers.Major && vers.Minor >= chvers.Minor
	}, nil
}
