package node

import (
	"fmt"
	"github.com/hashicorp/raft"
	"os"
	"path/filepath"
	"time"
)

// NewRaftNode 创建并启动 Raft 节点
func NewRaftNode(localID string, fsm raft.FSM) (*raft.Raft, error) {
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(localID)
	config.SnapshotInterval = 120 * time.Second
	config.SnapshotThreshold = 1024

	logStore := raft.NewInmemStore()

	stableStore := raft.NewInmemStore()

	snapshotStore, err := raft.NewFileSnapshotStore(filepath.Join("snapshots", string(localID)), 3, os.Stderr)
	if err != nil {
		return nil, err
	}

	_, transport := raft.NewInmemTransport(raft.ServerAddress(localID))

	r, err := raft.NewRaft(config, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return nil, fmt.Errorf("StudentNode.NewRaftNode error creating raft instance: %w", err)
	}

	configuration := raft.Configuration{
		Servers: []raft.Server{
			{
				ID:      config.LocalID,
				Address: transport.LocalAddr(),
			},
		},
	}
	r.BootstrapCluster(configuration)

	return r, nil
}
