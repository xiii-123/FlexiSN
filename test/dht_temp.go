package main

import (
	"context"
	"flag"
	golog "github.com/ipfs/go-log/v2"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"
)

// 默认的 ProtocolPrefix 和 Validator 配置
var defaultPrefix = "/default"

type blankValidator struct{}

func (blankValidator) Validate(_ string, _ []byte) error        { return nil }
func (blankValidator) Select(_ string, _ [][]byte) (int, error) { return 0, nil }

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// LibP2P code uses golog to log messages. They log with different
	// string IDs (i.e. "swarm"). We can control the verbosity level for
	// all loggers with:
	golog.SetAllLoggers(golog.LevelInfo) // Change to INFO for extra info

	// Parse options from the command line
	port := flag.Int("p", 0, "wait for incoming connections")
	target := flag.String("d", "", "target peer to dial")
	insecure := flag.Bool("insecure", false, "use an unencrypted connection")
	seed := flag.Int64("seed", 0, "set random seed for id generation")
	key := flag.String("k", "", "file to send (sender only)")
	value := flag.String("v", "", "value to send (sender only)")
	flag.Parse()

	if *port == 0 {
		log.Fatal("Please provide a port to bind on with -l")
	}

	// Make a host that listens on the given multiaddress
	host, err := newBasicHost(*port, *insecure, *seed)
	if err != nil {
		log.Fatal(err)
	}

	var kdht *dht.IpfsDHT

	baseOpts := []dht.Option{
		dht.ProtocolPrefix(protocol.ID(defaultPrefix)),
		dht.NamespacedValidator("v", blankValidator{}),
		dht.DisableAutoRefresh(),
	}

	if *target == "" {
		kdht, err = NewKDHT(ctx, host, []multiaddr.Multiaddr{}, append(baseOpts)...)
		if err != nil {
			log.WithField("error", err).Errorln("Create kademlia server failed.")
			return
		}
	} else {
		maddr, err := multiaddr.NewMultiaddr(*target)

		if err != nil {
			log.WithField("error", err).Errorln("Covert address to multiple address failed.")
			return
		}
		kdht, err = NewKDHT(ctx, host, []multiaddr.Multiaddr{maddr}, append(baseOpts)...)
		if err != nil {
			log.WithField("error", err).Errorln("Create kademlia server failed.")
			return
		}
	}

	if *target == "" {
		fullAddr := GetHostAddress(host)
		log.Printf("I am %s\n", fullAddr)
		log.Printf("Now, run dht.exe -p %d -d %s -k %s -v <value> on a different terminal.\n", *port+1, fullAddr, host.ID())
		log.Printf("Now, run dht.exe -p %d -d %s -k %s to get the value.\n", *port+1, fullAddr, host.ID())
		// Run until canceled.
		<-ctx.Done()
	} else if *key != "" && *value != "" {
		if *key == "" || *value == "" {
			log.Fatal("Please provide a k/v pair to send with -k and -v")
		}
		runPutValue(ctx, host, kdht, *key, *value)
	} else {
		runGetValue(ctx, host, kdht, *key)
	}
}

func runPutValue(ctx context.Context, ha host.Host, kdht *dht.IpfsDHT, key, value string) {
	fullAddr := GetHostAddress(ha)
	log.Printf("I am %s", fullAddr)

	log.Printf("key: %s, value: %s", key, value)
	err := kdht.PutValue(ctx, key, []byte(value))
	if err != nil {
		log.WithField("error", err).Errorln("Put value failed.")
		return
	}
	log.Println("Put value success.")

}

func runGetValue(ctx context.Context, ha host.Host, kdht *dht.IpfsDHT, key string) {
	fullAddr := GetHostAddress(ha)
	log.Printf("I am %s", fullAddr)

	value, err := kdht.GetValue(ctx, key)
	if err != nil {
		log.WithField("error", err).Errorln("Get value failed.")
		return
	}
	log.Println("Get value success: %s\n", string(value))
}

func NewKDHT(ctx context.Context, host host.Host, bootstrapPeers []multiaddr.Multiaddr, options ...dht.Option) (*dht.IpfsDHT, error) {
	// 如果没有引导节点，以服务器模式 ModeServer 启动
	if len(bootstrapPeers) == 0 {
		options = append(options, dht.Mode(dht.ModeServer))
		log.Infoln("Start node as a bootstrap server.")
	}

	// 生成一个 DHT 实例
	kdht, err := dht.New(ctx, host, options...)
	if err != nil {
		return nil, err
	}

	// 启动 DHT 服务
	if err = kdht.Bootstrap(ctx); err != nil {
		return nil, err
	}

	if len(bootstrapPeers) == 0 {
		return kdht, nil
	}

	// 遍历引导节点数组并尝试连接
	for _, peerAddr := range bootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		if err := host.Connect(ctx, *peerinfo); err != nil {
			log.Printf("Error while connecting to node %q: %-v", peerinfo, err)
			continue
		} else {
			log.Infof("Connection established with bootstrap node: %q",
				*peerinfo)
		}
	}

	return kdht, nil
}
