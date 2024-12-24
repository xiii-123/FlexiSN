package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	dht "main/DHT"
)

// parseData 定义结构体以匹配JSON格式，字段为字符串类型
type parseData struct {
	RootHash  string   `json:"rootHash"`
	RandomNum string   `json:"randomNum"`
	PublicKey string   `json:"publicKey"`
	Leaves    []string `json:"leaves"`
}

func main() {
	// 假设这是你收到的JSON字符串
	jsonStr := `{"type":"data","hash":"897a140edc97dc39663429f828b35c835c5eff03db0a46caf573adc0c743f9f9","height":"320815","address":"0a0f870f81376f77db1981f94f39b719f5eb3f7c","params":{"key":"565681","value":"{\"rootHash\": \"897a140edc97dc39663429f828b35c835c5eff03db0a46caf573adc0c743f9f9\",\"randomNum\": \"565681\",\"publicKey\": \"0a0f870f81376f77db1981f94f39b719f5eb3f7c\",\"leaves\": [\"1234\", \"4326\"]}"}}`

	// 定义结构体用于解析 JSON
	var data struct {
		Params struct {
			Value string `json:"value"`
		} `json:"params"`
	}

	// 解析 JSON 字符串
	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		log.Fatalf("Error unmarshalling JSON: %v", err)
	}
	fmt.Println("Extracted value:", data.Params.Value)

	// 创建parseData结构体的实例
	var parseData parseData

	// 解析JSON字符串到parseData结构体
	err = json.Unmarshal([]byte(data.Params.Value), &parseData)
	if err != nil {
		log.Fatalf("Error unmarshalling JSON: %v", err)
	}

	// 创建metaData结构体的实例
	var metaData dht.MetaData

	// 将字符串字段转换为字节数组
	metaData.RootHash, err = hex.DecodeString(parseData.RootHash)
	if err != nil {
		log.Fatalf("Error decoding rootHash: %v", err)
	}
	metaData.RandomNum, err = hex.DecodeString(parseData.RandomNum)
	if err != nil {
		log.Fatalf("Error decoding randomNum: %v", err)
	}
	metaData.PublicKey, err = hex.DecodeString(parseData.PublicKey)
	if err != nil {
		log.Fatalf("Error decoding publicKey: %v", err)
	}

	// 处理leaves字段
	metaData.Leaves = make([][]byte, len(parseData.Leaves))
	for i, leafStr := range parseData.Leaves {
		metaData.Leaves[i], err = hex.DecodeString(leafStr)
		if err != nil {
			log.Fatalf("Error decoding leaf: %v", err)
		}
	}

	// 打印转换后的结果
	fmt.Printf("RootHash: %x\n", metaData.RootHash)
	fmt.Printf("RandomNum: %x\n", metaData.RandomNum)
	fmt.Printf("PublicKey: %x\n", metaData.PublicKey)
	fmt.Println("Leaves:")
	for _, leaf := range metaData.Leaves {
		fmt.Printf("%x\n", leaf)
	}
}
