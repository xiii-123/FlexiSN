package main

import (
	"context"
	"flag"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
	"main/DHT"
	"os"
)

func main() {
	// 设置上下文和取消函数
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 解析命令行参数
	// Parse options from the command line
	port := flag.Int("p", 0, "wait for incoming connections")
	target := flag.String("d", "", "target peer to dial")
	key := flag.String("k", "", "file to send (sender only)")
	value := flag.String("v", "", "value to send (sender only)")
	flag.Parse()

	if *port == 0 {
		logrus.Fatal("Please provide a port to bind on with -l")
	}

	// 创建 DHT 配置
	dhtConfig := DHT.NewDHTConfig()
	dhtConfig.Port = *port

	if *target != "" {
		maddr, err := multiaddr.NewMultiaddr(*target)
		if err != nil {
			logrus.WithField("error", err).Errorln("Covert address to multiple address failed.")
			return
		}
		dhtConfig.BootstrapPeers = append(dhtConfig.BootstrapPeers, maddr)
	}

	// 创建 DHT 服务
	dhtService, err := DHT.NewDHTService(ctx, dhtConfig)
	if err != nil {
		logrus.Fatalf("Failed to create DHT service: %v", err)
	}
	dhtService.GetFileHandler(ctx, "data\\")
	dhtService.SendFileHandler(ctx)

	if *target == "" {
		fullAddr := DHT.GetHostAddress(dhtService.Host)
		logrus.Printf("I am %s\n", fullAddr)
		logrus.Printf("Now, go run testFileSwap.go -p %d -d %s -k %s on a different terminal.\n", *port+1, fullAddr, "hello")
		logrus.Printf("Now, go run testFileSwap.go -p %d -d %s -v %s to get the value.\n", *port+2, fullAddr, "hello.txt")
		// Run until canceled.
		<-ctx.Done()
	} else if *key != "" && *value == "" {
		logrus.Println("Here Send file")
		fullAddr := DHT.GetHostAddress(dhtService.Host)

		logrus.Printf("I am %s", fullAddr)

		logrus.Printf("key: %s", *key)

		maddr, err := multiaddr.NewMultiaddr(*target)
		if err != nil {
			logrus.WithField("error", err).Errorln("Covert address to multiple address failed.")
			return
		}
		file, err := os.Open(*key)
		if err != nil {
			logrus.WithError(err).Error("Can not open the file")
			return
		}
		err = dhtService.SendFile(ctx, maddr, *key, file)
		if err != nil {
			logrus.WithError(err).Error("Send file failed")
		} else {
			logrus.Println("Send file success")
		}
	} else {
		logrus.Println("Here Get File")
		fullAddr := DHT.GetHostAddress(dhtService.Host)
		logrus.Printf("I am %s", fullAddr)

		logrus.Printf("key: %s", *key)

		maddr, err := multiaddr.NewMultiaddr(*target)
		if err != nil {
			logrus.WithField("error", err).Errorln("Covert address to multiple address failed.")
			return
		}
		file, err := os.Open("data\\test.txt")
		if err != nil {
			logrus.WithError(err).Error("Can not open the file")
			return
		}
		err = dhtService.GetFile(ctx, maddr, "test.txt", "", file)
		if err != nil {
			logrus.WithError(err).Error("Can not get the file")
		}
	}
}
