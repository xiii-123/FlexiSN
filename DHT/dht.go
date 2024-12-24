package DHT

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"github.com/libp2p/go-libp2p-kad-dht"
	record "github.com/libp2p/go-libp2p-record"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
	"io"
	"strings"
)

const (
	AnnounceProtocol = "/Announce/1.0.0"
	LookupProtocol   = "/Lookup/1.0.0"
)

// 默认的 ProtocolPrefix 和 Validator 配置
var defaultPrefix = "/default"

type blankValidator struct{}

func (blankValidator) Validate(_ string, _ []byte) error        { return nil }
func (blankValidator) Select(_ string, _ [][]byte) (int, error) { return 0, nil }

type DHTService struct {
	Host   host.Host
	DHT    *dht.IpfsDHT
	Config *DHTConfig
}

type MetaData struct {
	RootHash  []byte   `json:"rootHash"`
	RandomNum []byte   `json:"randomNum"`
	PublicKey []byte   `json:"publicKey"`
	Leaves    [][]byte `json:"leaves"`
}

type DHTConfig struct {
	Port              int
	Insecure          bool
	Seed              int64
	BootstrapPeers    []multiaddr.Multiaddr
	ProtocolPrefix    string
	EnableAutoRefresh bool
	NameSpace         string
	Validator         record.Validator
}

// NewDHTConfig 返回一个包含默认配置的 DHTConfig 实例
// 返回值:
//   - DHTConfig: 包含默认配置的 DHTConfig 实例
func NewDHTConfig() DHTConfig {
	return DHTConfig{
		Port:              10000,
		Insecure:          false,
		Seed:              0,
		ProtocolPrefix:    defaultPrefix,
		EnableAutoRefresh: true,
		NameSpace:         "v",
		Validator:         blankValidator{}, // 使用默认的 blankValidator
	}
}

// NewDHTService 创建并启动 DHT 服务
// 参数:
//   - ctx: 上下文，用于控制生命周期
//   - config: DHT 配置
//
// 返回值:
//   - *DHTService: DHT 服务实例
//   - error: 错误信息
func NewDHTService(ctx context.Context, config DHTConfig) (*DHTService, error) {
	host, err := newBasicHost(config.Port, config.Insecure, config.Seed)
	if err != nil {
		return nil, xerrors.Errorf("failed to create host: %w", err)
	}

	kdht, err := newDHT(ctx, host, config)
	if err != nil {
		return nil, xerrors.Errorf("failed to create DHT instance: %w", err)
	}

	return &DHTService{
		Host:   host,
		DHT:    kdht,
		Config: &config,
	}, nil
}

// newDHT 创建一个 DHT 实例
// 参数:
//   - ctx: 上下文，用于控制生命周期
//   - host: 主机实例
//   - config: DHT 配置
//
// 返回值:
//   - *dht.IpfsDHT: DHT 实例
//   - error: 错误信息
func newDHT(ctx context.Context, host host.Host, config DHTConfig) (*dht.IpfsDHT, error) {
	opts := []dht.Option{
		dht.ProtocolPrefix(protocol.ID(config.ProtocolPrefix)),
		dht.NamespacedValidator(config.NameSpace, config.Validator),
	}

	if !config.EnableAutoRefresh {
		opts = append(opts, dht.DisableAutoRefresh())
	}

	// 如果没有引导节点，以服务器模式 ModeServer 启动
	//if len(config.BootstrapPeers) == 0 {
	opts = append(opts, dht.Mode(dht.ModeServer))
	logrus.Infoln("Start node as a bootstrap server.")
	//} else {
	//	opts = append(opts, dht.Mode(dht.ModeClient))
	//	logrus.Infoln("Start node as a client.")
	//}

	// 生成一个 DHT 实例
	kdht, err := dht.New(ctx, host, opts...)
	if err != nil {
		return nil, err
	}

	// 启动 DHT 服务
	if err = kdht.Bootstrap(ctx); err != nil {
		return nil, err
	}

	if len(config.BootstrapPeers) == 0 {
		return kdht, nil
	}

	// 遍历引导节点数组并尝试连接
	for _, peerAddr := range config.BootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		if err := host.Connect(ctx, *peerinfo); err != nil {
			logrus.Printf("Error while connecting to node %q: %-v", peerinfo, err)
			continue
		} else {
			logrus.Infof("Connection established with bootstrap node: %q",
				*peerinfo)
			kdht.RoutingTable().TryAddPeer(peerinfo.ID, true, true)
			peers := kdht.RoutingTable().ListPeers()
			logrus.Infof("RoutingTable size: %d", len(peers))
		}
	}

	return kdht, nil
}

// PutValue 向 DHT 中存储一个键值对
// 参数:
//   - ctx: 上下文，用于控制生命周期
//   - key: 键
//   - value: 值
//
// 返回值:
//   - error: 错误信息
func (d *DHTService) PutValue(ctx context.Context, key string, value []byte) error {
	key = "/" + d.Config.NameSpace + "/" + key
	err := d.DHT.PutValue(ctx, key, value)
	if err != nil {
		return xerrors.Errorf("failed to put value: %w", err)
	}
	logrus.Infof("Stored key-value pair: %s -> %s", key, value)
	return nil
}

// GetValue 从 DHT 中获取一个键值对
// 参数:
//   - ctx: 上下文，用于控制生命周期
//   - key: 键
//
// 返回值:
//   - string: 值
//   - error: 错误信息
func (d *DHTService) GetValue(ctx context.Context, key string) (string, error) {
	key = "/" + d.Config.NameSpace + "/" + key
	value, err := d.DHT.GetValue(ctx, key)
	if err != nil {
		return "", xerrors.Errorf("failed to get value: %w", err)
	}
	logrus.Infof("Retrieved value for key %s: %s", key, string(value))
	return string(value), nil
}

// Announce 向网络中的节点宣布一个 fileInfo
// 参数:
//   - ctx: 上下文，用于控制生命周期
//   - fileInfo: 要宣布的 fileInfo
//
// 返回值:
//   - error: 错误信息
func (d *DHTService) Announce(ctx context.Context, fileInfo string) error {
	peers, err := d.DHT.GetClosestPeers(ctx, fileInfo)
	if err != nil {
		return err
	}
	count := 0
	for _, p := range peers {
		s, err := d.Host.NewStream(ctx, p, AnnounceProtocol)
		if err != nil {
			logrus.Infof("Can not establish a stream with %d", p)
			continue
		}
		_, err = io.Copy(s, strings.NewReader(fileInfo+"\n"))
		if err != nil {
			logrus.Infof("Can not send chameHash with %d", p)
			continue
		}
		ai := peer.AddrInfo{
			ID:    d.Host.ID(),
			Addrs: d.Host.Addrs(),
		}
		buf, err := ai.MarshalJSON()
		_, err = io.Copy(s, bytes.NewReader(append(buf, []byte("\n")...)))
		if err != nil {
			logrus.Infof("Can not send host.ID with %d", p)
			continue
		}
		s.Close()
		count++
	}
	if count == 0 {
		return errors.New("No corresponding node can be found in the network")
	}
	return nil
}

// AnnounceHandler 处理 Announce 请求
// 参数:
//   - ctx: 上下文，用于控制生命周期
func (d *DHTService) AnnounceHandler(ctx context.Context) {
	host := d.Host
	dht := d.DHT
	host.SetStreamHandler(AnnounceProtocol, func(s network.Stream) {
		var err error
		buf := bufio.NewReader(s)

		str, err := buf.ReadString('\n')
		if err != nil {
			logrus.Fatalf("Can not read Announce fileInfo")
		}
		fileInfo := str
		fileInfo = strings.TrimRight(fileInfo, "\n")
		logrus.Infof("get fileInfo %s", fileInfo)

		str, err = buf.ReadString('\n')
		ai := peer.AddrInfo{}
		addrJson := []byte(str)[:len(str)-1]
		ai.UnmarshalJSON(addrJson)
		logrus.Infof("get addrInfo %s, %s", ai.ID, ai.Addrs)
		ps := dht.ProviderStore()
		err = ps.AddProvider(ctx, []byte(fileInfo), ai)
		if err != nil {
			// 使用WithError记录错误和堆栈跟踪
			logrus.WithError(err).Error("Can not Add Provider")
		}
		logrus.Infof("Add Provider success!")
		if err != nil {
			s.Reset()
		} else {
			s.Close()
		}
	})
}

// Lookup 找到持有对应 key 的所有节点
// 参数:
//   - ctx: 上下文，用于控制生命周期
//   - fileInfo: 要查找的 fileInfo
//
// 返回值:
//   - []multiaddr.Multiaddr: 节点地址列表
//   - error: 错误信息
func (d *DHTService) Lookup(ctx context.Context, fileInfo string) ([]peer.AddrInfo, error) {
	peers, err := d.DHT.GetClosestPeers(ctx, fileInfo)
	logrus.Infof("Find %d peers", len(peers))
	if err != nil {
		return nil, err
	}
	for _, p := range peers {
		s, err := d.Host.NewStream(ctx, p, LookupProtocol)
		if err != nil {
			logrus.Infof("Can not establish a stream with %d", p)
			continue
		}
		// 1, send a fileInfo
		_, err = io.Copy(s, strings.NewReader(fileInfo+"\n"))
		if err != nil {
			logrus.Infof("Can not send chameHash with %d", p)
			continue
		}
		logrus.Infof("send fileInfo success %s", fileInfo)

		buf := bufio.NewReader(s)

		// 2, read a bool
		str, err := buf.ReadString('\n')
		if err != nil {
			logrus.Fatalf("Can not read bool")
		}
		str = strings.TrimRight(str, "\n")
		if str != "true" {
			continue
		}
		logrus.Infof("read bool success %s", str)

		// 3, read addrIndo json
		var res []peer.AddrInfo
		for {
			str, err := buf.ReadString('\n')
			if err != nil && err != io.EOF || str == "" {
				logrus.Info(err)
				break
			}
			addrInfoJson := []byte(str)[:len(str)-1]
			if err != nil {
				logrus.WithError(err).Error("Can not read addrInfo")
			}
			logrus.Infof("read addrInfoJson success %b", addrInfoJson)

			ai := peer.AddrInfo{}
			err = ai.UnmarshalJSON(addrInfoJson)
			logrus.Infof("get addrInfo %s, %s, %s", ai.ID, ai.Addrs[0], ai.String())
			if err != nil {
				logrus.WithError(err).Error("Can not parse addrInfo")
			}
			res = append(res, ai)
		}
		s.Close()
		return res, nil
	}
	return nil, errors.New("The specified address was not found")
}

// addrInfosToMaddrs 将 AddrInfo 转换为 Multiaddr
// 参数:
//   - AddrInfos: AddrInfo 列表
//
// 返回值:
//   - []multiaddr.Multiaddr: Multiaddr 列表
//   - error: 错误信息
func addrInfosToMaddrs(AddrInfos []peer.AddrInfo) ([]multiaddr.Multiaddr, error) {
	var res []multiaddr.Multiaddr
	for _, ai := range AddrInfos {
		for _, maddr := range ai.Addrs {
			temp, err := multiaddr.NewMultiaddr(maddr.String() + "/p2p/" + ai.ID.String())
			if err != nil {
				logrus.Infof("Can not convert AddrInfo to Multiaddr")
				return nil, err
			}
			res = append(res, temp)
		}
	}
	return res, nil
}

// LookupHandler 处理 Lookup 请求
// 参数:
//   - ctx: 上下文，用于控制生命周期
func (d *DHTService) LookupHandler(ctx context.Context) {
	host := d.Host
	dht := d.DHT
	host.SetStreamHandler(LookupProtocol, func(s network.Stream) {
		var err error
		// 1, read fileInfo
		buf := bufio.NewReader(s)
		str, err := buf.ReadString('\n')
		if err != nil {
			logrus.Fatalf("Can not read Announce id")
		}
		fileInfo := str
		fileInfo = strings.TrimRight(fileInfo, "\n")
		logrus.Printf("get fileInfo success %s", fileInfo)

		// 2, send bool
		ps := dht.ProviderStore()
		peers, err := ps.GetProviders(ctx, []byte(fileInfo))
		if err != nil {
			logrus.WithError(err).Error("")
		}
		logrus.Printf("find %d peers", len(peers))

		if len(peers) == 0 {
			s.Write([]byte("false" + "\n"))
			return
		}
		s.Write([]byte("true" + "\n"))
		logrus.Println("send bool success")

		// 3, send multiaddr
		for _, p := range peers {
			res, err := p.MarshalJSON()

			res = append(res, []byte("\n")...)
			if err != nil {
				logrus.Info(err)
			}
			_, err = s.Write(res)
			if err != nil {
				logrus.WithError(err).Error("Can not send addrInfo")
			}
			logrus.Printf("send multiaddr success %b", res[:len(res)-1])
		}

		if err != nil {
			s.Reset()
		} else {
			s.Close()
		}
	})
}
