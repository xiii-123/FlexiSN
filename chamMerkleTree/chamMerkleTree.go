package chamMerkleTree

import (
	"bytes"
	"container/list"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	dht "main/DHT"
	"math/big"
	"os"
)

// MerkleConfig 包含Merkle树的配置信息
type MerkleConfig struct {
	BlockSize int // 文件分块大小，默认为4MB
}

// ChameleomPubKey 包含Chameleon哈希的公钥
type ChameleomPubKey struct {
	pubX *big.Int
	pubY *big.Int
}

func (pubKey *ChameleomPubKey) Serialize() []byte {
	return append(pubKey.pubX.Bytes(), pubKey.pubY.Bytes()...)
}

func DeserializeChameleomPubKey(data []byte) *ChameleomPubKey {
	pubXBytes := data[:32]
	pubYBytes := data[32:]
	return &ChameleomPubKey{
		pubX: new(big.Int).SetBytes(pubXBytes),
		pubY: new(big.Int).SetBytes(pubYBytes),
	}
}

// ChameleonRandomNum 包含Chameleon哈希的随机数
type ChameleonRandomNum struct {
	rX *big.Int
	rY *big.Int
	s  *big.Int
}

func (randomNum *ChameleonRandomNum) Serialize() []byte {
	return append(append(randomNum.rX.Bytes(), randomNum.rY.Bytes()...), randomNum.s.Bytes()...)
}

func DeserializeChameleonRandomNum(data []byte) *ChameleonRandomNum {
	rXBytes := data[:32]
	rYBytes := data[32:64]
	sBytes := data[64:]
	return &ChameleonRandomNum{
		rX: new(big.Int).SetBytes(rXBytes),
		rY: new(big.Int).SetBytes(rYBytes),
		s:  new(big.Int).SetBytes(sBytes),
	}
}

// GenerateChameleonKeyPair 生成Chameleon哈希的公私钥对
func GenerateChameleonKeyPair() ([]byte, *ChameleomPubKey) {
	priv, pubX, pubY, _ := elliptic.GenerateKey(GetCurve(), rand.Reader)
	return priv, &ChameleomPubKey{
		pubX: pubX,
		pubY: pubY,
	}
}

// NewMerkleConfig 创建一个新的Merkle树配置
func NewMerkleConfig() *MerkleConfig {
	return &MerkleConfig{
		BlockSize: 4 * 1024 * 1024, // 4MB
	}
}

// MerkleNode 表示Merkle树的节点
type MerkleNode struct {
	Hash  []byte
	Left  *MerkleNode
	Right *MerkleNode
}

// getHash 计算给定数据的SHA-256哈希
func getHash(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// VerifyMerkleRoot 验证Merkle树的根节点是否正确
func VerifyMerkleRoot(text, hX []byte, pubKey *ChameleomPubKey, randomNum *ChameleonRandomNum) bool {
	return VerifyHash(text, randomNum.rX, randomNum.rY, randomNum.s, pubKey.pubX, pubKey.pubY, new(big.Int).SetBytes(hX))
}

// BuildMerkleTree 构建一个Merkle树，并返回根节点和一个Chameleon随机数。
// 参数:
// - filePath: 文件路径，表示要读取的文件。
// - config: Merkle树的配置，包括块大小等信息。
// - pubKey: Chameleon哈希的公钥。
// 返回值:
// - *MerkleNode: Merkle树的根节点。
// - *ChameleonRandomNum: Chameleon随机数。
// - []byte: 计算chameleon hash的消息。
// - error: 如果发生错误，返回错误信息。
//
// 该函数首先打开指定文件，并将文件内容按块大小读取，创建叶子节点。
// 然后通过两两合并节点的方式，逐层构建Merkle树，直到只剩下一个或两个节点。
// 最后，计算根节点的哈希值，并返回根节点和Chameleon随机数。
func BuildMerkleTree(file *os.File, config *MerkleConfig, pubKey *ChameleomPubKey) (*MerkleNode, *ChameleonRandomNum, []byte, error) {
	// 读取文件并创建叶子节点
	var nodes []*MerkleNode
	buffer := make([]byte, config.BlockSize)
	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, nil, nil, err
		}
		if n == 0 {
			break
		}
		node := &MerkleNode{Hash: getHash(buffer[:n])}
		nodes = append(nodes, node)
		// 如果读取的数据量小于块大小，说明已到达文件末尾
		if n < config.BlockSize {
			break
		}
	}
	// 构建Merkle树
	for len(nodes) > 2 {
		var newLevel []*MerkleNode
		for i := 0; i < len(nodes); i += 2 {
			var hash []byte
			if i+1 < len(nodes) {
				hash = getHash(append(nodes[i].Hash, nodes[i+1].Hash...))
				newLevel = append(newLevel, &MerkleNode{
					Hash:  hash,
					Left:  nodes[i],
					Right: nodes[i+1],
				})
			} else {
				// 如果是最后一个节点，直接复制
				newLevel = append(newLevel, nodes[i])
			}
		}
		nodes = newLevel
	}

	var combined []byte
	var root *MerkleNode
	// 返回根节点
	if len(nodes) == 1 {
		combined = nodes[0].Hash
		root = &MerkleNode{
			Left: nodes[0],
		}
	} else {
		combined = append(nodes[0].Hash, nodes[1].Hash...)
		root = &MerkleNode{
			Left:  nodes[0],
			Right: nodes[1],
		}
	}
	rX, rY, s, hX := ComputeHash(combined, pubKey.pubX, pubKey.pubY)
	root.Hash = hX.Bytes()

	return root, &ChameleonRandomNum{
		rX: rX,
		rY: rY,
		s:  s,
	}, combined, nil
}

// UpdateMerkleTree 更新Merkle树
//
// 该函数读取指定文件的内容，并根据文件内容构建Merkle树。然后，它使用给定的Chameleon哈希密钥和随机数
// 更新Merkle树的根节点，并返回新的根节点和新的随机数。
//
// 参数:
// - filePath: 文件路径
// - config: Merkle树的配置，包括块大小等信息
// - pubKey: Chameleon哈希的公钥
// - secKey: Chameleon哈希的私钥
// - prevRoot: 之前的Merkle树根节点的哈希值
// - chameleonHash: 计算Chameleon哈希的消息
// - randomNum: 当前的Chameleon随机数
//
// 返回值:
// - *MerkleNode: 新的Merkle树根节点
// - *ChameleonRandomNum: 新的Chameleon随机数
// - error: 如果发生错误，返回错误信息
func UpdateMerkleTree(file *os.File, config *MerkleConfig, pubKey *ChameleomPubKey, secKey, prevRootHash, chameleonHash []byte, randomNum *ChameleonRandomNum) (*MerkleNode, *ChameleonRandomNum, error) {
	// 读取文件并创建叶子节点
	var nodes []*MerkleNode
	buffer := make([]byte, config.BlockSize)
	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, nil, err
		}
		if n == 0 {
			break
		}
		node := &MerkleNode{Hash: getHash(buffer[:n])}
		nodes = append(nodes, node)
		// 如果读取的数据量小于块大小，说明已到达文件末尾
		if n < config.BlockSize {
			break
		}
	}
	// 构建Merkle树
	for len(nodes) > 2 {
		var newLevel []*MerkleNode
		for i := 0; i < len(nodes); i += 2 {
			var hash []byte
			if i+1 < len(nodes) {
				hash = getHash(append(nodes[i].Hash, nodes[i+1].Hash...))
				newLevel = append(newLevel, &MerkleNode{
					Hash:  hash,
					Left:  nodes[i],
					Right: nodes[i+1],
				})
			} else {
				// 如果是最后一个节点，直接复制
				newLevel = append(newLevel, nodes[i])
			}
		}
		nodes = newLevel
	}

	var combined []byte
	var root *MerkleNode
	// 返回根节点
	if len(nodes) == 1 {
		combined = nodes[0].Hash
		root = &MerkleNode{
			Left: nodes[0],
		}
	} else {
		combined = append(nodes[0].Hash, nodes[1].Hash...)
		root = &MerkleNode{
			Left:  nodes[0],
			Right: nodes[1],
		}
	}

	newRX, newRY, newS := FindCollision(chameleonHash, randomNum.rX, randomNum.rY, randomNum.s, new(big.Int).SetBytes(prevRootHash), combined, secKey)
	root.Hash = prevRootHash

	return root, &ChameleonRandomNum{
		rX: newRX,
		rY: newRY,
		s:  newS,
	}, nil
}

// LevelOrderTraversal 层序遍历Merkle树并打印结构
func LevelOrderTraversal(root *MerkleNode) {
	if root == nil {
		return
	}

	queue := list.New()
	queue.PushBack(root)

	for queue.Len() > 0 {
		levelSize := queue.Len()
		for i := 0; i < levelSize; i++ {
			node := queue.Remove(queue.Front()).(*MerkleNode)
			fmt.Printf("%x ", node.Hash[:8])
			if node.Left != nil {
				queue.PushBack(node.Left)
			}
			if node.Right != nil {
				queue.PushBack(node.Right)
			}
		}
		fmt.Println() // 打印完一层后换行
	}
}

// getAllLeaves 从Merkle树的根节点获取所有叶子节点的哈希值
func GetAllLeavesHashes(root *MerkleNode) [][]byte {
	var leafHashes [][]byte
	if root == nil {
		return leafHashes
	}

	// 使用队列进行层序遍历
	queue := list.New()
	queue.PushBack(root)

	for queue.Len() > 0 {
		node := queue.Remove(queue.Front()).(*MerkleNode)
		// 如果是叶子节点，则添加到leafHashes列表中
		if node.Left == nil && node.Right == nil {
			leafHashes = append(leafHashes, node.Hash)
		}
		// 将子节点加入队列
		if node.Left != nil {
			queue.PushBack(node.Left)
		}
		if node.Right != nil {
			queue.PushBack(node.Right)
		}
	}

	return leafHashes
}

// GenerateMerkleProof 函数生成给定目标节点的默克尔证明路径。
// 它使用深度优先搜索（DFS）算法从根节点开始遍历默克尔树，
// 并收集从根节点到目标节点路径上的所有兄弟节点的哈希值。
//
// 参数:
// - root: 指向默克尔树根节点的指针。
// - target: 目标节点的哈希值。
//
// 返回值:
// 返回一个二维字节数组，其中包含从根节点到目标节点路径上所有兄弟节点的哈希值。
//
// 示例:
// 假设有一个默克尔树，其结构如下:
//
//	     root
//	    /    \
//	  A        B
//	 / \      / \
//	C   D    E   F
//
// 如果目标节点是 D，则返回的兄弟节点哈希值数组为 [[][B][C][]]。其中，空格表示路径上此高度节点所在的位置。
func GenerateMerkleProof(root *MerkleNode, target []byte) [][]byte {
	var path []*MerkleNode
	var siblings [][]byte

	// dfs 是深度优先搜索的辅助函数
	var dfs func(node *MerkleNode) bool
	dfs = func(node *MerkleNode) bool {
		if node == nil {
			return false
		}
		// 将当前节点加入路径
		path = append(path, node)
		// 如果找到了目标节点
		if bytes.Equal(node.Hash, target) {
			// 遍历路径，收集兄弟节点的值
			for i := 0; i < len(path)-1; i++ {
				parent := path[i]
				// 初始化兄弟节点为空
				sibling := [][]byte{{}, {}}
				// 当前节点是左子节点，则右子节点是兄弟节点
				if parent.Right != nil && parent.Right != path[i+1] {
					sibling[1] = parent.Right.Hash
				}
				// 当前节点是右子节点，则左子节点是兄弟节点
				if parent.Left != nil && parent.Left != path[i+1] {
					sibling[0] = parent.Left.Hash
				}
				// 将兄弟节点添加到结果中
				siblings = append(siblings, sibling...)
			}
			return true
		}
		// 继续搜索左子树和右子树
		if dfs(node.Left) || dfs(node.Right) {
			return true
		}
		// 回溯，移除当前节点
		path = path[:len(path)-1]
		return false
	}

	// 从根节点开始搜索
	dfs(root)
	return siblings
}

// VerifyMerkleProof 验证给定的 Merkle 证明是否有效。
// 参数：
// - rootHash: Merkle 树的根哈希值。
// - targetHash: 目标哈希值，即需要验证的叶子节点哈希。
// - merkleProof: Merkle 证明路径，包含从叶子节点到根节点的哈希值。
// - pubKey: 公钥，用于验证 Chameleon 哈希。
// - randomNum: 随机数，用于验证 Chameleon 哈希。
// 返回值：
// - bool: 如果证明有效，返回 true；否则返回 false。
func VerifyMerkleProof(rootHash, targetHash []byte, merkleProof [][]byte, pubKey *ChameleomPubKey, randomNum *ChameleonRandomNum) bool {
	currentHash := targetHash
	for i := len(merkleProof) - 2; i >= 2; i -= 2 {
		var combined []byte
		if len(merkleProof[i]) == 0 {
			combined = append(currentHash, merkleProof[i+1]...)
		} else {
			combined = append(merkleProof[i], currentHash...)
		}
		currentHash = getHash(combined)
		fmt.Printf("currentHash: %x\n", currentHash)
	}
	var combined []byte
	if len(merkleProof[0]) == 0 && len(merkleProof[1]) == 0 {
		combined = currentHash
	} else if len(merkleProof[0]) != 0 {
		combined = append(merkleProof[0], currentHash...)
	} else {
		combined = append(currentHash, merkleProof[1]...)
	}

	toBe := VerifyMerkleRoot(combined, rootHash, pubKey, randomNum)

	return toBe
}

func RebuildMerkleTreeFromMetaData(metaData *dht.MetaData) (*MerkleNode, *ChameleonRandomNum, *ChameleomPubKey, error) {
	// 读取metaData并创建叶子节点
	var nodes []*MerkleNode
	for _, leaf := range metaData.Leaves {
		node := &MerkleNode{Hash: leaf}
		nodes = append(nodes, node)
	}
	// 构建Merkle树
	for len(nodes) > 2 {
		var newLevel []*MerkleNode
		for i := 0; i < len(nodes); i += 2 {
			var hash []byte
			if i+1 < len(nodes) {
				hash = getHash(append(nodes[i].Hash, nodes[i+1].Hash...))
				newLevel = append(newLevel, &MerkleNode{
					Hash:  hash,
					Left:  nodes[i],
					Right: nodes[i+1],
				})
			} else {
				// 如果是最后一个节点，直接复制
				newLevel = append(newLevel, nodes[i])
			}
		}
		nodes = newLevel
	}

	var combined []byte
	var root *MerkleNode
	// 返回根节点
	if len(nodes) == 1 {
		combined = nodes[0].Hash
		root = &MerkleNode{
			Left: nodes[0],
		}
	} else {
		combined = append(nodes[0].Hash, nodes[1].Hash...)
		root = &MerkleNode{
			Left:  nodes[0],
			Right: nodes[1],
		}
	}

	randomNum := DeserializeChameleonRandomNum(metaData.RandomNum)
	pubKey := DeserializeChameleomPubKey(metaData.PublicKey)
	toBe := VerifyMerkleRoot(combined, metaData.RootHash, pubKey, randomNum)
	if !toBe {
		return nil, nil, nil, fmt.Errorf("Merkle root verification failed")
	}
	root.Hash = metaData.RootHash

	return root, randomNum, pubKey, nil
}

func main() {
	// 使用默认配置构建Merkle树
	config := NewMerkleConfig()
	config.BlockSize = 4 * 1024

	// 创建公私钥
	secKey, pubKey := GenerateChameleonKeyPair()
	fmt.Printf("privateKey: %x\n", secKey)
	file, err := os.Open("C:\\Users\\admin\\Desktop\\云平台账号.txt")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}

	// 根据公钥生成原始chameleon hash
	root, randomNum, chameleonHash, err := BuildMerkleTree(file, config, pubKey)
	if err != nil {
		fmt.Println("Error building Merkle tree:", err)
		return
	}
	fmt.Printf("root hash: %x\n", root.Hash)

	// 遍历merkle tree
	LevelOrderTraversal(root)

	targetHash, _ := hex.DecodeString("f1ac93290110a4e0ef724b77f0b28f653fc858635d23a8c5bb415129196c8e7e")

	// 生成merkle proof
	proofs := GenerateMerkleProof(root, targetHash)
	fmt.Println("merkle proof: ", proofs)
	// 验证merkle proof
	fmt.Println("verify merkle proof:", VerifyMerkleProof(root.Hash, targetHash, proofs, pubKey, randomNum))

	// 更新文件，重建merkle tree
	newRoot, newRandomNum, _ := UpdateMerkleTree(file, config, pubKey, secKey, root.Hash, chameleonHash, randomNum)
	newProofs := GenerateMerkleProof(newRoot, targetHash)
	// 验证新的merkle proof
	fmt.Println("verify new merkle proof:", VerifyMerkleProof(newRoot.Hash, targetHash, newProofs, pubKey, newRandomNum))

}
