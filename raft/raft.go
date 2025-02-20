package raft

import (
	"github.com/hashicorp/raft"
	"memoryDataBase/interfaces"
	"memoryDataBase/raft/fsm"
	"memoryDataBase/raft/node"
)

// RaftInitializer 定义 Raft 初始化接口
type RaftInitializer interface {
	InitRaft(localID string, service interfaces.StudentServiceInterface) (*raft.Raft, error)
}

// RaftInitializerImpl 实现 RaftInitializer 接口
type RaftInitializerImpl struct{}

// InitRaft 初始化 Raft
func (r *RaftInitializerImpl) InitRaft(localID string, service interfaces.StudentServiceInterface) (*raft.Raft, error) {
	fsmInstance := fsm.NewStudentFSM(service)
	return node.NewRaftNode(localID, fsmInstance)
}
