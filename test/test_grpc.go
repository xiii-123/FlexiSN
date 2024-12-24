package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"main/rpc" // 引入 blockchainclient 包
)

// 定义people结构体
type People struct {
	Name    string `json:"name"`
	Age     int    `json:"age"`
	Gender  string `json:"gender"`
	Address string `json:"address"`
}

func main() {
	// 创建 gRPC 客户端
	client, err := rpc.NewClient("localhost:45555")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close() // 在结束时关闭连接

	// 获取区块编号
	blockNumberResp, err := client.GetBlockNumber(context.Background())
	if err != nil {
		log.Fatalf("Failed to get block number: %v", err)
	}
	fmt.Printf("Block Number: %d, Timestamp: %d\n", blockNumberResp.GetNumber(), blockNumberResp.GetTimestamp())

	// 获取区块信息
	hash := "89fa10e01b40a99e3685d07983e3b4e895827dd2a0a4a862740e296563d3b243"
	transResp, err := client.GetTransactionByHash(context.Background(), hash)
	if err != nil {
		log.Fatalf("Failed to get transaction by hash: %s,  %v", hash, err)
	}
	fmt.Printf("Block Hash: %s, Height: %d, Body: %s\n", transResp.Body.GetHash(), transResp.Body.GetHeight(), transResp.Body.GetData())

	// 发送带数据的交易
	// 创建一个People实例
	people := People{
		Name:    "张三",
		Age:     30,
		Gender:  "男",
		Address: "北京市朝阳区",
	}

	// 将People实例转换为JSON字符串
	jsonData, err := json.Marshal(people)
	if err != nil {
		fmt.Println("转换失败:", err)
		return
	}
	txResp, err := client.SendTransactionWithData(context.Background(), "set", "0a0f870f81376f77db1981f94f39b719f5eb3f7c", "565681", string(jsonData))
	if err != nil {
		log.Fatalf("Failed to send transaction with data: %v", err)
	}
	fmt.Printf("Transaction Hash: %s\n", txResp.GetTxHash())
}
