package cmd

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
	"io"
	dht "main/DHT"
	"main/chamMerkleTree"
	"main/manager"
	"main/run"
	"os"
	"path/filepath"
)

func init() {
	run.RegisterCommand(run.Command{
		Name:        "get",
		Description: "Gets a file from network",
		Action:      getAction,
	})
}

func getAction(ctx context.Context, params map[string]string) error {
	fileName, exists := params["-f"]
	if !exists {
		logrus.Printf("Please provide a file name with -f")
		return run.NoRequiredParamError
	}
	filePath, exists := params["-path"]
	if !exists {
		filePath = "data"
	}

	dhtService := manager.GetDHTService()

	// 1, Get the file information from the blockchain
	root, _, _, err := getChameleonMerkleTree(fileName)
	if err != nil {
		return nil
	}
	logrus.Infof("Get the root hash %s", hex.EncodeToString(root.Hash))

	// 2, get the file splits from the network
	leaves := chamMerkleTree.GetAllLeavesHashes(root)
	files := []*os.File{}
	for _, leaf := range leaves {
		// get the file split from the network
		splitName := hex.EncodeToString(leaf)
		peers, err := dhtService.DHT.GetClosestPeers(ctx, hex.EncodeToString(leaf))
		if err != nil {
			logrus.Errorf("Get closest peers failed")
			return err
		}
		if len(peers) == 0 {
			peers = dhtService.DHT.RoutingTable().ListPeers()
			logrus.Infof("bootstrap peers", len(peers))
		}
		logrus.Infof("Get closest peers success")

		// create a temp files
		tempFile, err := os.CreateTemp("", splitName)
		var find bool
		if err != nil {
			return err
		}
		for _, peer := range peers {

			addrInfo, err := dhtService.DHT.FindPeer(ctx, peer)
			if err != nil {
				return err
			}
			maddrs := addrInfo.Addrs
			maddr, err := multiaddr.NewMultiaddr(maddrs[0].String() + "/p2p/" + peer.String())
			if err != nil {
				return err
			}
			err = dhtService.GetFile(ctx, maddr, splitName, "", tempFile)
			if err != nil {
				logrus.Println("Get file failed", err)
			} else {
				find = true
				files = append(files, tempFile)
				break
			}

		}
		if !find {
			return errors.New(fmt.Sprintf("Can not find the file split %s", splitName))
		}
	}

	// 3, merge the file splits into the original file
	filePath = filepath.Join(filePath, fileName)
	err = mergeFiles(files, filePath)
	if err != nil {
		return err
	}

	// 4, remove the temp files
	for _, file := range files {
		file.Close()
		os.Remove(file.Name())
	}

	// 5, Announce the file to the network
	dhtService.Announce(ctx, fileName)

	return nil
}

func getChameleonMerkleTree(fileHash string) (*chamMerkleTree.MerkleNode, *chamMerkleTree.ChameleonRandomNum, *chamMerkleTree.ChameleomPubKey, error) {
	// 1, get information from db
	var metaData dht.MetaData
	err := manager.GetDBManager().LoadFromMemory(fileHash, &metaData)
	if err != nil {
		logrus.Errorf("Load metadata from db failed: %v", err)
		return nil, nil, nil, err
	}

	// 2, rebuild the chameleon merkle tree
	root, randomNum, pubKey, err := chamMerkleTree.RebuildMerkleTreeFromMetaData(&metaData)
	if err != nil {
		return nil, nil, nil, err
	}
	return root, randomNum, pubKey, nil
}

func mergeFiles(fileList []*os.File, targetFilePath string) error {
	// 创建目标文件
	targetFile, err := os.Create(targetFilePath)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer targetFile.Close()

	// 遍历每个临时文件并将其内容写入目标文件
	for _, file := range fileList {
		// 确保每个文件都被正确打开
		if file == nil {
			continue
		}

		// 将文件的内容拷贝到目标文件
		_, err := file.Seek(0, io.SeekStart) // 确保从文件开头开始读取
		if err != nil {
			return fmt.Errorf("failed to seek in file: %w", err)
		}

		_, err = io.Copy(targetFile, file)
		if err != nil {
			return fmt.Errorf("failed to copy data from temp file to target file: %w", err)
		}
	}

	return nil
}
