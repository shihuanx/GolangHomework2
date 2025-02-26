package node

import (
	"fmt"
	"log"
	"memoryDataBase/config"
	"memoryDataBase/interfaces"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
)

// NewRaftNode 创建并启动 Raft 节点
func NewRaftNode(node config.Node, peers []*config.Peer, fsm raft.FSM, service interfaces.StudentServiceInterface) (*raft.Raft, error) {
	log.Printf("开始创建 Raft 节点: NodeID=%s, Address=%s", node.NodeId, node.Address)

	// 配置 Raft
	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(node.NodeId)
	raftConfig.SnapshotInterval = 120 * time.Second
	raftConfig.SnapshotThreshold = 1024

	// 初始化存储
	logStore := raft.NewInmemStore()
	stableStore := raft.NewInmemStore()

	// 为每个节点创建独立的快照目录
	snapshotDir := filepath.Join("snapshots", node.NodeId)
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return nil, fmt.Errorf("创建快照目录失败: NodeID=%s, Directory=%s, Error=%w", node.NodeId, snapshotDir, err)
	}
	snapshotStore, err := raft.NewFileSnapshotStore(snapshotDir, 3, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("创建快照存储失败: NodeID=%s, Error=%w", node.NodeId, err)
	}

	// 初始化传输层 通过TCP传输
	transport, err := raft.NewTCPTransport(node.Address, nil, 3, 10*time.Second, os.Stderr)
	if err != nil {
		log.Printf("创建 Raft 传输层失败: NodeID=%s Address=%s Error=%v", node.NodeId, node.Address, err)
		return nil, err
	}
	if transport == nil {
		log.Printf("创建 Raft 传输层返回 nil: NodeID=%s Address=%s", node.NodeId, node.Address)
		return nil, fmt.Errorf("创建 Raft 传输层返回 nil")
	}
	log.Printf("创建 Raft 传输层成功: NodeID=%s Address=%s transport=%v", node.NodeId, node.Address, transport)

	// 创建 Raft 实例
	r, err := raft.NewRaft(raftConfig, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return nil, fmt.Errorf("创建 Raft 实例失败: NodeID：%s, Error：%w", node.NodeId, err)
	}

	// 如果是第一个节点，初始化集群
	if len(peers) == 0 {
		log.Printf("节点 %s 是第一个节点，开始初始化集群", node.NodeId)
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raft.ServerID(node.NodeId),
					Address: raft.ServerAddress(node.Address),
				},
			},
		}
		future := r.BootstrapCluster(configuration)
		if err := future.Error(); err != nil {
			return nil, fmt.Errorf("节点 %s 初始化集群失败: %w", node.NodeId, err)
		}
		log.Printf("节点 %s 集群初始化成功", node.NodeId)
	} else {
		log.Printf("节点 %s 尝试加入现有集群，等待选举完成...", node.NodeId)

		leaderPortAddr, err := service.GetLeaderPortAddr()
		if err != nil {
			log.Printf("节点：%s获取leader地址失败：%v", node.NodeId, err)
			return nil, fmt.Errorf("节点：%s获取leader地址失败：%w", node.NodeId, err)
		}

		url := fmt.Sprintf("http://localhost:%s/JoinRaftCluster?nodeID=%s&nodeAddress=%s&portAddress=%s", leaderPortAddr, node.NodeId, node.Address, node.PortAddress)
		_, err = http.Get(url)
		if err != nil {
			log.Printf("节点：%s加入集群失败：%v", node.NodeId, err)
			return nil, fmt.Errorf("节点：%s加入集群失败：%w", node.NodeId, err)
		}
	}
	return r, nil
}
