package test

// import (
// 	//"github.com/samuel/go-zookeeper/zk"
// 	"github.com/go-zookeeper/zk"
// 	"github.com/go-zookeeper/zk/zktest"

// 	"testing"
// 	"time"
// )

// var zkServer *zktest.ZkTestServer

// func TestMyZooKeeperIntegration(t *testing.T) {
// 	// Start an embedded ZooKeeper instance
// 	zkServer, _, err := zktest.StartTestCluster(1, t, nil, nil)
// 	if err != nil {
// 		t.Fatalf("Failed to start ZooKeeper: %v", err)
// 	}
// 	defer zkServer.Stop()

// 	// Connect to the embedded ZooKeeper instance
// 	zkConn, _, err := zk.Connect([]string{zkServer.Addr()}, time.Second)
// 	if err != nil {
// 		t.Fatalf("Failed to connect to ZooKeeper: %v", err)
// 	}
// 	defer zkConn.Close()

// 	//
// }
