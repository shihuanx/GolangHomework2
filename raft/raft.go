package raft

import (
	"github.com/hashicorp/raft"
	"log"
	"node2/config"
	"node2/interfaces"
	"node2/raft/fsm"
	nodepkg "node2/raft/node"
)

// RaftInitializerImpl 实现 Raft 初始化器接口
type RaftInitializerImpl struct{}

// InitRaft 初始化 Raft 节点
func (r *RaftInitializerImpl) InitRaft(node config.Node, peers []*config.Peer, service interfaces.StudentServiceInterface) (*raft.Raft, error) {
	log.Printf("开始初始化 Raft 节点: NodeID=%s, Address=%s", node.NodeId, node.Address)
	fsmInstance := fsm.NewStudentFSM(service)
	raftNode, err := nodepkg.NewRaftNode(node, peers, fsmInstance, service)
	if err != nil {
		log.Printf("初始化 Raft 节点失败: NodeID=%s, Error=%v", node.NodeId, err)
		return nil, err
	}
	log.Printf("Raft 节点初始化成功: NodeID=%s", node.NodeId)
	return raftNode, nil
}
