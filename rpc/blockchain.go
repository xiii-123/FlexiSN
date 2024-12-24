package rpc

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"main/rpc/pb" // 引入生成的 pb 包
)

// BlockchainClient 封装 gRPC 客户端
type BlockchainClient struct {
	client pb.BlockchainClient
	conn   *grpc.ClientConn
}

// NewClient 创建一个新的 gRPC 客户端连接
func NewClient(address string) (*BlockchainClient, error) {
	// 连接到 gRPC 服务
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}

	// 创建客户端
	client := pb.NewBlockchainClient(conn)
	return &BlockchainClient{client: client, conn: conn}, nil
}

// Close 关闭 gRPC 客户端连接
func (bc *BlockchainClient) Close() {
	if bc.conn != nil {
		err := bc.conn.Close()
		if err != nil {
			log.Printf("failed to close connection: %v", err)
		}
	}
}

// GetBlockNumber 获取区块编号
func (bc *BlockchainClient) GetBlockNumber(ctx context.Context) (*pb.BlockNumberResp, error) {
	// 调用 GetBlockNumber 方法
	resp, err := bc.client.GetBlockNumber(ctx, &emptypb.Empty{})
	if err != nil {
		// 捕获错误状态并返回
		if grpcErr, ok := status.FromError(err); ok {
			if grpcErr.Code() == codes.NotFound {
				return nil, fmt.Errorf("block number not found: %v", grpcErr.Message())
			}
		}
		return nil, fmt.Errorf("failed to get block number: %v", err)
	}
	return resp, nil
}

func (bc *BlockchainClient) GetBlockByNumber(ctx context.Context, number uint64) (*pb.GetBlockResp, error) {
	// 创建请求
	req := &pb.GetBlockReq{
		Number: &number,
	}

	// 调用 GetBlockByNumber 方法
	resp, err := bc.client.GetBlockByNumber(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get block by number: %v", err)
	}
	return resp, nil
}

// GetBlockByHash 根据区块哈希获取区块
func (bc *BlockchainClient) GetBlockByHash(ctx context.Context, hash string) (*pb.GetBlockResp, error) {
	// 创建请求
	req := &pb.GetBlockReq{
		Hash: &hash,
	}

	// 调用 GetBlockByHash 方法
	resp, err := bc.client.GetBlockByHash(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get block by hash: %v", err)
	}
	return resp, nil
}

// GetTransactionByHash 根据交易哈希获取交易
func (bc *BlockchainClient) GetTransactionByHash(ctx context.Context, hash string) (*pb.GetTransactionResp, error) {
	// 创建请求
	req := &pb.GetTransactionReq{
		Hash: &hash,
	}

	// 调用 GetTransactionByHash 方法
	resp, err := bc.client.GetTransactionByHash(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction by hash: %v", err)
	}
	return resp, nil
}

// SendTransactionWithData 发送带数据的交易
func (bc *BlockchainClient) SendTransactionWithData(ctx context.Context, txType, receiver, key, value string) (*pb.SendTransactionWithDataResp, error) {
	// 创建请求
	req := &pb.SendTransactionWithDataReq{
		Type:     &txType,
		Receiver: &receiver,
		Key:      &key,
		Value:    &value,
	}

	// 调用 SendTransactionWithData 方法
	resp, err := bc.client.SendTransactionWithData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction with data: %v", err)
	}
	return resp, nil
}
