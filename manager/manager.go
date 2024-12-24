package manager

import (
	"context"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
	dht "main/DHT"
	"main/chamMerkleTree"
	"main/db"
	"main/rpc"
	"time"
)

type Parameters struct {
	SecKey []byte
	PubKey *chamMerkleTree.ChameleomPubKey
}

var (
	DHTService *dht.DHTService

	GRPCClient *rpc.BlockchainClient

	DBManager *db.DBManager

	Params *Parameters
)

func InitDHTService(ctx context.Context, port int, target string) error {
	var err error

	dhtConfig := dht.NewDHTConfig()
	dhtConfig.Port = port

	if target != "" {
		maddr, err := multiaddr.NewMultiaddr(target)
		if err != nil {
			logrus.WithField("error", err).Errorln("Covert address to multiple address failed.")
			return err
		}
		dhtConfig.BootstrapPeers = append(dhtConfig.BootstrapPeers, maddr)
	}

	DHTService, err = dht.NewDHTService(ctx, dhtConfig)
	if err != nil {
		return err
	}
	logrus.Println("I am ", dht.GetHostAddress(DHTService.Host))
	DHTService.AnnounceHandler(ctx)
	DHTService.LookupHandler(ctx)
	DHTService.SendFileHandler(ctx, "data")
	DHTService.GetFileHandler(ctx, "data")
	return nil
}

func GetDHTService() *dht.DHTService {
	return DHTService
}

func InitGRPCClient(address string) error {
	var err error
	GRPCClient, err = rpc.NewClient(address)
	if err != nil {
		return err
	}
	return nil
}

func GetGRPCClient() *rpc.BlockchainClient {
	return GRPCClient
}

func InitDBManager(dbFile string) error {
	var err error
	DBManager, err = db.NewDBManager(dbFile)
	if err != nil {
		return err
	}
	go DBManager.PeriodicSave(10 * time.Minute)
	return nil
}

func GetDBManager() *db.DBManager {
	return DBManager
}

func InitParameters(secKey, pubKey []byte) {
	Params = &Parameters{
		SecKey: secKey,
		PubKey: chamMerkleTree.DeserializeChameleomPubKey(pubKey),
	}
}

func GetParameters() *Parameters {
	return Params
}
