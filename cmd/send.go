package cmd

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
	"io"
	"main/DHT"
	"main/chamMerkleTree"
	"main/manager"
	"main/run"
	"os"
	"strconv"
)

func init() {
	run.RegisterCommand(run.Command{
		Name:        "send",
		Description: "Sends a file to network",
		Action:      sendAction,
	})
}

// todo: use memoFile instead of tempFIle
func sendAction(ctx context.Context, params map[string]string) error {
	filePath, exists := params["-f"]
	if !exists {
		logrus.Printf("Please provide a file path with -f")
		return run.NoRequiredParamError
	}
	numString, exists := params["-n"]
	num := 5
	var err error
	if exists {
		num, err = strconv.Atoi(numString)
		if err != nil {
			return err
		}
	}

	dhtService := manager.GetDHTService()
	parameter := manager.GetParameters()

	// 1, Generate Chameleon Merkle tree
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	logrus.Infof("Send file %s", filePath)
	defer file.Close()

	pubKey := parameter.PubKey
	config := chamMerkleTree.NewMerkleConfig()
	root, randomNum, _, err := chamMerkleTree.BuildMerkleTree(file, config, pubKey)
	if err != nil {
		return err
	}
	_, err = file.Seek(0, 0)
	fileBuf := bufio.NewReader(file)

	// 2, Send metadata to the network
	err = sendMetadata(ctx, root, randomNum, pubKey)
	if err != nil {
		return err
	}
	logrus.Infof("Send metadata %s", hex.EncodeToString(root.Hash))

	// 3, Send the file splits to the network
	// todo: use multiThreads
	leaves := chamMerkleTree.GetAllLeavesHashes(root)
	buffer := make([]byte, config.BlockSize)
	for _, leaf := range leaves {

		splitName := hex.EncodeToString(leaf)
		logrus.Infof("Send split %s", splitName)

		n, err := fileBuf.Read(buffer)
		if err != nil && err != io.EOF {
			logrus.Errorf("Read file failed")
			return err
		}
		if n == 0 {
			logrus.Infof("Read file finished")
			break
		}
		logrus.Infof("Read fileSplit success")

		// create temp file and write buffer to it
		tempFile, err := os.CreateTemp("", splitName)
		if err != nil {
			logrus.Errorf("Create temp file failed")
			return err
		}
		_, err = tempFile.Write(buffer[:n])
		if err != nil {
			logrus.Errorf("Write buffer to temp file failed")
			return err
		}
		logrus.Infof("Write buffer to temp file success")

		peers, err := dhtService.DHT.GetClosestPeers(ctx, splitName)
		if err != nil {
			logrus.Errorf("Get closest peers failed")
			return err
		}
		if len(peers) == 0 {
			peers = dhtService.DHT.RoutingTable().ListPeers()
			logrus.Infof("bootstrap peers", len(peers))
		}
		logrus.Infof("Get closest peers success")

		numTemp := num
		for _, peer := range peers {
			tempFile.Seek(0, 0)
			logrus.Infof("Send split %s to %s", splitName, peer)
			if numTemp == 0 {
				break
			}
			numTemp--
			addrInfo, err := dhtService.DHT.FindPeer(ctx, peer)
			if err != nil {
				return err
			}
			maddrs := addrInfo.Addrs
			maddr, err := multiaddr.NewMultiaddr(maddrs[0].String() + "/p2p/" + peer.String())
			if err != nil {
				logrus.Errorf("Convert address to multiaddress failed")
				return err
			}
			logrus.Infof("Send split %s to %s", splitName, maddr)

			// send file
			err = dhtService.SendFile(ctx, maddr, splitName, tempFile)
			if err != nil {
				logrus.Errorf("Send split %s to %s failed", splitName, peer)
				return err
			}
			logrus.Infof("Send split %s to %s success", splitName, peer)
			dhtService.Announce(ctx, splitName)

			// remove temp file
			tempFile.Close()
			os.Remove(tempFile.Name())

			// 如果读取的数据量小于块大小，说明已到达文件末尾
			if n < config.BlockSize {
				break
			}

		}

	}
	// 4, Announce the file to the network
	//dhtService.Announce(ctx, hex.EncodeToString(root.Hash))

	return nil
}

// send metadata to norn
func sendMetadata(ctx context.Context, root *chamMerkleTree.MerkleNode, randomNum *chamMerkleTree.ChameleonRandomNum, pubKey *chamMerkleTree.ChameleomPubKey) error {
	// 1, Serialize the metadata
	metaData := &DHT.MetaData{
		RootHash:  root.Hash,
		RandomNum: randomNum.Serialize(),
		PublicKey: pubKey.Serialize(),
		Leaves:    chamMerkleTree.GetAllLeavesHashes(root),
	}
	// 2, Send the metadata to the network
	// 将结构体转换为 JSON 字符串
	jsonData, err := json.Marshal(metaData)
	if err != nil {
		logrus.Errorf("Error marshalling struct:", err)
		return err
	}

	// 3, Send the metadata to the network
	_, err = manager.GetGRPCClient().SendTransactionWithData(ctx, "set", hex.EncodeToString(root.Hash), "metadata", string(jsonData))
	if err != nil {
		logrus.Errorf("Send metadata to network failed")
		return err
	}

	// 4, storage locally
	err = manager.GetDBManager().SaveToMemory(hex.EncodeToString(root.Hash), metaData)
	if err != nil {
		logrus.Errorf("Save metadata to memory failed")
		return err
	}

	//// 5, verify storage
	//var metaDataVerify DHT.MetaData
	//manager.GetDBManager().LoadFromMemory(hex.EncodeToString(root.Hash), &metaDataVerify)
	//
	//logrus.Infof("Verify metadata success", metaDataVerify.RootHash)

	return nil
}
