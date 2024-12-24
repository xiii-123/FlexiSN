package DHT

import (
	"crypto/rand"
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/multiformats/go-multiaddr"
	"io"
	mrand "math/rand"
)

// makeBasicHost creates a LibP2P host with a random peer ID listening on the
// given multiaddress. It won't encrypt the connection if insecure is true.
func newBasicHost(listenPort int, insecure bool, randseed int64) (host.Host, error) {
	var r io.Reader
	if randseed == 0 {
		r = rand.Reader
	} else {
		r = mrand.New(mrand.NewSource(randseed))
	}

	// Generate a key pair for this host. We will use it at least
	// to obtain a valid host ID.
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil, err
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)),
		libp2p.Identity(priv),
		libp2p.DisableRelay(),
	}

	if insecure {
		opts = append(opts, libp2p.NoSecurity)
	}

	return libp2p.New(opts...)
}

func GetHostAddress(host host.Host) string {
	// Build host multiaddress
	hostAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", host.ID()))

	// Now we can build a full multiaddress to reach this host
	// by encapsulating both addresses:
	addr := host.Addrs()[0]
	return addr.Encapsulate(hostAddr).String()
}
