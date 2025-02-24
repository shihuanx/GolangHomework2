package raft

import (
	"github.com/hashicorp/raft"
	"log"
	"memoryDataBase/interfaces"
	"memoryDataBase/raft/fsm"
	"memoryDataBase/raft/node"
)

// RaftInitializerImpl 实现 Raft 初始化器接口
type RaftInitializerImpl struct{}

// InitRaft 初始化 Raft 节点
func (r *RaftInitializerImpl) InitRaft(nodeID string, address string, peers []string, service interfaces.StudentServiceInterface) (*raft.Raft, error) {
	log.Printf("开始初始化 Raft 节点: NodeID=%s, Address=%s", nodeID, address)
	fsmInstance := fsm.NewStudentFSM(service)
	raftNode, err := node.NewRaftNode(nodeID, address, peers, fsmInstance)
	if err != nil {
		log.Printf("初始化 Raft 节点失败: NodeID=%s, Error=%v", nodeID, err)
		return nil, err
	}
	log.Printf("Raft 节点初始化成功: NodeID=%s", nodeID)
	return raftNode, nil
}
