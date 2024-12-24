package main

import (
	"encoding/hex"
	"fmt"
	"gopkg.in/yaml.v3"
	"main/chamMerkleTree"
	"os"
)

// 配置结构体
type Config struct {
	SecKey string `yaml:"SecKey"`
	PubKey string `yaml:"PubKey"`
}

func generateConfig(filename string) error {
	secKey, pubKey := chamMerkleTree.GenerateChameleonKeyPair()
	// 配置数据，包含密钥
	config := Config{
		SecKey: hex.EncodeToString(secKey),             // 示例 SecKey
		PubKey: hex.EncodeToString(pubKey.Serialize()), // 示例 PubKey
	}
	fmt.Printf("%s\n%s\n", config.SecKey, config.PubKey)

	// 将配置结构体转换为 YAML 格式
	data, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("error marshaling config: %v", err)
	}

	// 创建并写入 config.yml 文件
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("error writing to file: %v", err)
	}

	fmt.Printf("Config file %s has been generated.\n", filename)
	return nil
}

func main() {
	// 生成配置文件 config.yml
	err := generateConfig("config.yml")
	if err != nil {
		fmt.Println("Error generating config:", err)
	}
}
