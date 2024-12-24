package websocket

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	dht "main/DHT"
)

// Define a struct to map the JSON structure
type Params struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Data struct {
	Type    string `json:"type"`
	Hash    string `json:"hash"`
	Height  string `json:"height"`
	Address string `json:"address"`
	Params  Params `json:"params"`
}

// parseData 定义结构体以匹配JSON格式，字段为字符串类型
type parseData struct {
	RootHash  string   `json:"rootHash"`
	RandomNum string   `json:"randomNum"`
	PublicKey string   `json:"publicKey"`
	Leaves    []string `json:"leaves"`
}

func ParseTxValue(jsonStr string) (*dht.MetaData, error) {
	// Unmarshal the JSON string into the Data struct
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

	return &metaData, nil
}
