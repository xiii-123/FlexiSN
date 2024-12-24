package main

//
//import (
//	"context"
//	"flag"
//	"github.com/multiformats/go-multiaddr"
//	"github.com/sirupsen/logrus"
//)
//
//func main() {
//	// 设置上下文和取消函数
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	// 解析命令行参数
//	// Parse options from the command line
//	port := flag.Int("p", 0, "wait for incoming connections")
//	target := flag.String("d", "", "target peer to dial")
//	key := flag.String("k", "", "file to send (sender only)")
//	value := flag.String("v", "", "value to send (sender only)")
//	flag.Parse()
//
//	if *port == 0 {
//		logrus.Fatal("Please provide a port to bind on with -l")
//	}
//
//	// 创建 DHT 配置
//	dhtConfig := NewDHTConfig()
//	dhtConfig.Port = *port
//
//	if *target != "" {
//		maddr, err := multiaddr.NewMultiaddr(*target)
//		if err != nil {
//			logrus.WithField("error", err).Errorln("Covert address to multiple address failed.")
//			return
//		}
//		dhtConfig.BootstrapPeers = append(dhtConfig.BootstrapPeers, maddr)
//	}
//
//	// 创建 DHT 服务
//	dhtService, err := NewDHTService(ctx, dhtConfig)
//	if err != nil {
//		logrus.Fatalf("Failed to create DHT service: %v", err)
//	}
//	dhtService.AnnounceHandler(ctx)
//	dhtService.LookupHandler(ctx)
//
//	if *target == "" {
//		fullAddr := GetHostAddress(dhtService.Host)
//		logrus.Printf("I am %s\n", fullAddr)
//		logrus.Printf("Now, run testPutGet.exe -p %d -d %s -k /v/%s -v 123 on a different terminal.\n", *port+1, fullAddr, dhtService.Host.ID())
//		logrus.Printf("Now, run testPutGet.exe -p %d -d %s -k /v/%s to get the value.\n", *port+1, fullAddr, dhtService.Host.ID())
//		// Run until canceled.
//		<-ctx.Done()
//	} else if *key != "" && *value != "" {
//		fullAddr := GetHostAddress(dhtService.Host)
//		logrus.Printf("I am %s", fullAddr)
//
//		logrus.Printf("key: %s, value: %s", key, value)
//
//		err = dhtService.PutValue(ctx, *key, []byte(*value))
//		if err != nil {
//			logrus.Fatalf("can not put the k/v", err)
//		}
//	} else {
//		fullAddr := GetHostAddress(dhtService.Host)
//		logrus.Printf("I am %s", fullAddr)
//
//		logrus.Printf("key: %s", key)
//
//		value, err := dhtService.GetValue(ctx, *key)
//		if err != nil {
//			logrus.Fatalf("can not find the value", err)
//		}
//		logrus.Infof("find the value success!", value)
//	}
//}
