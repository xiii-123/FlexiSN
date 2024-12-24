package run

import (
	"bufio"
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"log"
	"main/manager"
	"main/websocket"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

// 配置结构体
type Config struct {
	SecKey string `yaml:"SecKey"`
	PubKey string `yaml:"PubKey"`
}

// Command 结构体定义
type Command struct {
	Name        string
	Description string
	Action      func(context.Context, map[string]string) error
}

var NoRequiredParamError = fmt.Errorf("Required command line parameters are missing!")

// 全局变量
var (
	// 注册命令的全局map
	commands = make(map[string]Command)
	mu       sync.Mutex
)

// 将 16 进制字符串解码为 []byte
func decodeFromHex(data string) ([]byte, error) {
	return hex.DecodeString(data)
}

// 导入配置文件并返回配置结构体
func importConfig(filename string) error {
	// 读取文件内容
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("error reading file: %v", err)
	}

	// 解析 YAML 内容到 Config 结构体
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return fmt.Errorf("error unmarshaling config: %v", err)
	}

	// 解码 SecKey 和 PubKey
	configSecKey, err := decodeFromHex(config.SecKey)
	if err != nil {
		return fmt.Errorf("error decoding SecKey: %v", err)
	}
	configPubKey, err := decodeFromHex(config.PubKey)
	if err != nil {
		return fmt.Errorf("error decoding PubKey: %v", err)
	}

	// 更新配置结构体中的 SecKey 和 PubKey 为 []byte
	manager.InitParameters(configSecKey, configPubKey)

	return nil
}

// 注册命令
func RegisterCommand(cmd Command) {
	mu.Lock()
	defer mu.Unlock()
	commands[cmd.Name] = cmd
}

// 显示帮助信息
func showHelp() {
	mu.Lock()
	defer mu.Unlock()
	fmt.Println("Available commands:")
	for _, cmd := range commands {
		fmt.Printf("%s - %s\n", cmd.Name, cmd.Description)
	}
}

// 启动并运行交互式命令行
func Start() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 导入配置文件
	err := importConfig("config.yml")
	if err != nil {
		fmt.Println("Error importing config:", err)
		return
	}

	//解析命令行参数
	port := flag.Int("p", 0, "wait for incoming connections")
	target := flag.String("d", "", "target peer to dial")
	flag.Parse()
	if *port == 0 {
		logrus.Fatal("Please provide a port to bind on with -l")
	}

	// 创建 DHT 服务
	err = manager.InitDHTService(ctx, *port, *target)
	if err != nil {
		logrus.Fatalf("Failed to create DHT service: %v", err)
	}

	// 创建GRPC client
	err = manager.InitGRPCClient("localhost:45555")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// 运行websocket订阅norn中的消息
	go websocket.RunWebSocket(ctx)

	// 创建 DBManager
	err = manager.InitDBManager("./db/kvstore.db")
	if err != nil {
		log.Fatal("Error initializing DBManager:", err)
	}

	// 欢迎信息
	logrus.Println("Welcome to the Interactive CLI!")
	logrus.Println("Type 'help' for a list of commands.")

	// 创建输入扫描器
	scanner := bufio.NewScanner(os.Stdin)
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-interrupt:
			logrus.Println("Received interrupt, shutting down...")
			return
		default: // 显示提示符
			fmt.Print("> ")

			// 读取用户输入
			scanner.Scan()
			input := scanner.Text()
			input = strings.TrimSpace(input)

			// 输入为空则继续等待
			if input == "" {
				continue
			}

			// 判断是否为帮助命令
			if input == "help" {
				showHelp()
				continue
			}

			// 解析输入的命令和参数
			cmdName, params := parseInput(input)

			// 查找并执行命令
			mu.Lock()
			cmd, exists := commands[cmdName]
			mu.Unlock()
			if exists {
				err := cmd.Action(ctx, params)
				if err != nil {
					logrus.Println("Error:", err)
				}
			} else {
				logrus.Println("Unknown command:", input)
			}
		}

	}
}

// 解析命令行输入，分离命令和参数
func parseInput(input string) (string, map[string]string) {
	parts := strings.Fields(input)
	cmd := parts[0]
	params := make(map[string]string)

	// 如果命令后有参数，解析参数
	if len(parts) > 1 {
		for i := 1; i < len(parts); i++ {
			if strings.HasPrefix(parts[i], "-") {
				// 如果参数后面有值，才处理
				if i+1 < len(parts) && !strings.HasPrefix(parts[i+1], "-") {
					params[parts[i]] = parts[i+1]
					i++ // 跳过下一个值
				} else {
					// 没有值的参数也加入map，值为空字符串
					params[parts[i]] = ""
				}
			}
		}
	}

	return cmd, params
}
