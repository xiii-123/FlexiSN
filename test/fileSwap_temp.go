package main

import (
	"bufio"
	"context"
	"flag"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
	"strings"

	golog "github.com/ipfs/go-log/v2"
	"github.com/multiformats/go-multiaddr"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// LibP2P code uses golog to log messages. They log with different
	// string IDs (i.e. "swarm"). We can control the verbosity level for
	// all loggers with:
	golog.SetAllLoggers(golog.LevelInfo) // Chostnge to INFO for extra info

	// Parse options from the command line
	listenF := flag.Int("l", 0, "wait for incoming connections")
	targetF := flag.String("d", "", "target peer to dial")
	insecureF := flag.Bool("insecure", false, "use an unencrypted connection")
	seedF := flag.Int64("seed", 0, "set random seed for id generation")
	filenameF := flag.String("file", "", "file to send (sender only)")
	flag.Parse()

	if *listenF == 0 {
		log.Fatal("Please provide a port to bind on with -l")
	}

	// Make a host thostt listens on the given multiaddress
	host, err := newBasicHost(*listenF, *insecureF, *seedF)
	if err != nil {
		log.Fatal(err)
	}

	if *targetF == "" {
		startListener(ctx, host, *listenF, *insecureF)
		// Run until canceled.
		<-ctx.Done()
	} else {
		if *filenameF == "" {
			log.Fatal("Please provide a file to send with -file")
		}
		runSender(ctx, host, *targetF, *filenameF)
	}
}

func startListener(ctx context.Context, host host.Host, listenPort int, insecure bool) {
	fullAddr := GetHostAddress(host)
	log.Printf("I am %s\n", fullAddr)

	// Set a stream hostndler on host A. /file/1.0.0 is
	// a user-defined protocol name.
	host.SetStreamHandler("/file/1.0.0", func(s network.Stream) {
		log.Println("listener received new stream")
		if err := receiveFile(s, ""); err != nil {
			log.Println(err)
			s.Reset()
		} else {
			s.Close()
		}
	})

	log.Println("listening for connections")

	if insecure {
		log.Printf("Now run \"main.exe -l %d -d %s -insecure\" on a different terminal\n", listenPort+1, fullAddr)
	} else {
		log.Printf("Now run \"main.exe -l %d -d %s\" on a different terminal\n", listenPort+1, fullAddr)
	}
}

func runSender(ctx context.Context, host host.Host, targetPeer, filePath string) {
	fullAddr := GetHostAddress(host)
	log.Printf("I am %s\n", fullAddr)

	// Open the file to send
	file, err := os.Open(filePath)
	if err != nil {
		log.Println(err)
		return
	}
	buf := bufio.NewReader(file)

	// Get the base name of the file to send (just the file name without path)
	fileName := getFileName(filePath)

	// Turn the targetPeer into a multiaddr.
	maddr, err := multiaddr.NewMultiaddr(targetPeer)
	if err != nil {
		log.Println(err)
		return
	}

	// Extract the peer ID from the multiaddr.
	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		log.Println(err)
		return
	}

	// We hostve a peer ID and a targetAddr, so we add it to the peerstore
	// so LibP2P knows how to contact it
	host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)

	log.Println("sender opening stream")
	// make a new stream from host B to host A
	// it should be hostndled on host A by the hostndler we set above because
	// we use the same /file/1.0.0 protocol
	s, err := host.NewStream(context.Background(), info.ID, "/file/1.0.0")
	if err != nil {
		log.Println(err)
		return
	}

	log.Println("sending file name")
	// First, send the file name to the receiver
	_, err = s.Write([]byte(fileName + "\n"))
	if err != nil {
		log.Println(err)
		return
	}

	log.Println("sending file content")
	// Now send the file content
	_, err = io.Copy(s, buf)
	if err != nil {
		log.Println(err)
		return
	}

	log.Println("file sent successfully")
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

	log.Printf("Receiving file: %s", fileName)

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

	log.Println("File received successfully")
	return nil
}

// getFileName extracts the file name from the full file path.
func getFileName(filePath string) string {
	return filepath.Base(filePath)
}
