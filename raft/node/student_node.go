package node

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/raft"
)

// 定义全局互斥锁
var snapshotDirMutex sync.Mutex

// NewRaftNode 创建并启动 Raft 节点
func NewRaftNode(nodeID string, address string, peers []string, fsm raft.FSM) (*raft.Raft, error) {
	log.Printf("开始创建 Raft 节点: NodeID=%s, Address=%s", nodeID, address)

	// 配置 Raft
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(nodeID)
	config.SnapshotInterval = 120 * time.Second
	config.SnapshotThreshold = 1024

	// 初始化存储
	logStore := raft.NewInmemStore()
	stableStore := raft.NewInmemStore()

	// 为每个节点创建独立的快照目录
	snapshotDirMutex.Lock()
	snapshotDir := filepath.Join("snapshots", nodeID)
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		snapshotDirMutex.Unlock()
		return nil, fmt.Errorf("创建快照目录失败: NodeID=%s, Directory=%s, Error=%w", nodeID, snapshotDir, err)
	}
	snapshotStore, err := raft.NewFileSnapshotStore(snapshotDir, 3, os.Stderr)
	snapshotDirMutex.Unlock()
	if err != nil {
		return nil, fmt.Errorf("创建快照存储失败: NodeID=%s, Error=%w", nodeID, err)
	}

	// 初始化传输层 通过TCP传输
	transport, err := raft.NewTCPTransport(address, nil, 3, 10*time.Second, os.Stderr)
	if err != nil {
		log.Printf("创建 Raft 传输层失败: NodeID=%s Address=%s Error=%v", nodeID, address, err)
		return nil, err
	}
	if transport == nil {
		log.Printf("创建 Raft 传输层返回 nil: NodeID=%s Address=%s", nodeID, address)
		return nil, fmt.Errorf("创建 Raft 传输层返回 nil")
	}
	log.Printf("创建 Raft 传输层成功: NodeID=%s Address=%s transport=%v", nodeID, address, transport)

	// 创建 Raft 实例
	r, err := raft.NewRaft(config, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return nil, fmt.Errorf("创建 Raft 实例失败: NodeID：%s, Error：%w", nodeID, err)
	}

	// 如果是第一个节点，初始化集群
	if len(peers) == 0 {
		log.Printf("节点 %s 是第一个节点，开始初始化集群", nodeID)
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raft.ServerID(nodeID),
					Address: raft.ServerAddress(address),
				},
			},
		}
		future := r.BootstrapCluster(configuration)
		if err := future.Error(); err != nil {
			return nil, fmt.Errorf("节点 %s 初始化集群失败: %w", nodeID, err)
		}
		log.Printf("节点 %s 集群初始化成功", nodeID)
	} else {
		log.Printf("节点 %s 尝试加入现有集群，等待选举完成...", nodeID)
		time.Sleep(10 * time.Second) // 增加等待时间到 10 秒

		url := fmt.Sprintf("http://localhost:8080/JoinRaftCluster?nodeID=%s&nodeAddress=%s", nodeID, address)
		_, err := http.Get(url)
		if err != nil {
			log.Printf("节点：%s加入集群失败：%v", nodeID, err)
			return nil, fmt.Errorf("节点：%s加入集群失败：%w", nodeID, err)
		}
	}
	return r, nil
}
