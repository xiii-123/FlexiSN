package DHT

import (
	"bufio"
	"context"
	"errors"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	pro "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	sendFileProtocol = "/SendFile/1.0.0"
	getFileProtocol  = "/GetFile/1.0.0"
)

// SendFile 将文件发送到目标节点。
// 参数:
// - ctx: 上下文，用于控制取消操作。
// - target: 目标节点的多地址。
// - filePath: 要发送的文件路径。
// 返回值:
// - error: 如果发送过程中出现错误，则返回错误信息。
func (d *DHTService) SendFile(ctx context.Context, target multiaddr.Multiaddr, fileName string, file io.ReadWriter) error {
	host := d.Host

	// Extract peer ID and add to peerstore
	info, err := peer.AddrInfoFromP2pAddr(target)
	if err != nil {
		return err
	}
	host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)

	// Use the common file transfer handler
	return d.handleFileTransfer(ctx, info.ID, sendFileProtocol, fileName, file)
}

// GetFile 从目标节点检索文件。
// 参数:
// - ctx: 上下文，用于控制取消操作。
// - target: 目标节点的多地址。
// - fileInfo: 要检索的文件信息。
// - path: 文件保存路径。
// 返回值:
// - error: 如果检索过程中出现错误，则返回错误信息。
func (d *DHTService) GetFile(ctx context.Context, target multiaddr.Multiaddr, fileInfo, path string, file io.ReadWriter) error {
	host := d.Host

	// Extract peer ID and add to peerstore
	info, err := peer.AddrInfoFromP2pAddr(target)
	if err != nil {
		return err
	}
	host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)

	// Use the common file transfer handler
	return d.handleFileTransfer(ctx, info.ID, getFileProtocol, fileInfo, file)
}

// handleFileTransfer 处理通过流发送和接收文件。
// 参数:
// - ctx: 上下文，用于控制取消操作。
// - target: 目标节点的ID。
// - protocol: 使用的协议。
// - fileName: 文件名。
// - file: 文件读取器，如果是发送文件则传入文件读取器，否则传入nil。
// 返回值:
// - error: 如果传输过程中出现错误，则返回错误信息。
func (d *DHTService) handleFileTransfer(ctx context.Context, target peer.ID, protocol, fileName string, file io.ReadWriter) error {
	host := d.Host

	// Open a stream to the target peer
	s, err := host.NewStream(ctx, target, pro.ID(protocol))
	if err != nil {
		return err
	}
	defer s.Close()

	// Send the file name
	if _, err := s.Write([]byte(fileName + "\n")); err != nil {
		return err
	}

	// Send or receive the file content
	if protocol == sendFileProtocol {
		// Sending file
		buf := bufio.NewReader(file)
		if _, err := io.Copy(s, buf); err != nil {
			return err
		}
		logrus.Println("File sent successfully")
	} else {
		// Receiving file

		// Read the response about file availability
		responseBuf := bufio.NewReader(s)
		str, err := responseBuf.ReadString('\n')
		if err != nil {
			return err
		}
		str = strings.TrimSpace(str)
		if str != "true" {
			logrus.Printf("Peer does not have the file %s", fileName)
			return errors.New("peer does not have the file")
		}
		logrus.Printf("Peer has the file %s", fileName)

		buf := bufio.NewWriter(file)

		// Copy the incoming stream to the output file
		// Ensure all data is copied before closing the stream
		if _, err := io.Copy(buf, s); err != nil {
			logrus.Printf("Cannot receive the file %s", fileName)
			return err
		}

		// Data copy is complete, now we can close the stream.
		logrus.Println("File received successfully")
	}

	return nil
}

// SendFileHandler 监听传入的文件请求。
// 参数:
// - ctx: 上下文，用于控制取消操作。
func (d *DHTService) SendFileHandler(ctx context.Context, path string) {
	host := d.Host
	host.SetStreamHandler(sendFileProtocol, func(s network.Stream) {
		logrus.Println("Received new stream")
		if err := receiveFile(s, path); err != nil {
			logrus.Println(err)
			s.Reset()
		} else {
			s.Close()
		}
	})
	logrus.Println("Listening for connections")
}

// GetFileHandler 监听传入的文件请求以发送文件。
// 参数:
// - ctx: 上下文，用于控制取消操作。
// - path: 文件存储路径。
func (d *DHTService) GetFileHandler(ctx context.Context, path string) {
	host := d.Host
	host.SetStreamHandler(getFileProtocol, func(s network.Stream) {
		defer s.Close()
		buf := bufio.NewReader(s)

		// Get fileInfo from the incoming request
		str, err := buf.ReadString('\n')
		if err != nil {
			logrus.Fatalf("Cannot read fileInfo: %v", err)
		}
		fileInfo := strings.TrimSpace(str)
		logrus.Printf("Requested file: %s", fileInfo)

		// Attempt to find the file
		file, err := os.Open(filepath.Join(path, fileInfo))
		if err != nil {
			s.Write([]byte("false\n"))
			logrus.Printf("Cannot find the file %s", fileInfo)
			return
		}
		defer file.Close()

		// Confirm file availability
		s.Write([]byte("true\n"))
		logrus.Printf("File found: %s", fileInfo)

		// Send the file
		fbuf := bufio.NewReader(file)
		if _, err := io.Copy(s, fbuf); err != nil {
			logrus.Fatal(err)
			return
		}
		logrus.Printf("File send success: %s", fileInfo)
	})
}

// receiveFile 从流中接收文件并写入磁盘。
// 参数:
// - s: 网络流。
// - path: 文件保存路径。
// 返回值:
// - error: 如果接收过程中出现错误，则返回错误信息。
func receiveFile(s network.Stream, path string) error {
	buf := bufio.NewReader(s)

	// Read the file name
	fileName, err := buf.ReadString('\n')
	if err != nil {
		return err
	}
	fileName = strings.TrimSpace(fileName)

	logrus.Printf("Receiving file: %s", fileName)

	// Prepare the output file path
	if path != "" {
		fileName = filepath.Join(path, fileName)
	}

	// Create the output file
	outFile, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Copy the incoming stream to the output file
	if _, err := io.Copy(outFile, s); err != nil {
		return err
	}

	logrus.Println("File received successfully")
	return nil
}

// getFileName extracts the file name from the full file path.
func getFileName(filePath string) string {
	return filepath.Base(filePath)
}
