package main

import (
	"encoding/json"
	"fmt"
)

type MetaData struct {
	RootHash  []byte   `json:"rootHash"`
	RandomNum []byte   `json:"randomNum"`
	PublicKey []byte   `json:"publicKey"`
	Leaves    [][]byte `json:"leaves"`
}

func main() {
	// 示例数据
	metaData := MetaData{
		RootHash:  []byte("root-hash-value"),
		RandomNum: []byte("random-number-value"),
		PublicKey: []byte("public-key-value"),
		Leaves:    [][]byte{[]byte("leaf1"), []byte("leaf2")},
	}

	// 将结构体转换为 JSON 字符串
	jsonData, err := json.Marshal(metaData)
	if err != nil {
		fmt.Println("Error marshalling struct:", err)
		return
	}

	// 打印 JSON 字符串
	fmt.Println(string(jsonData))
}
