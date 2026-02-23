package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	csipb "github.com/container-storage-interface/spec/lib/go/csi"
	kcsisvc "github.com/rachanaanugandula/kube-pfs/pkg/csi"
	"github.com/rachanaanugandula/kube-pfs/pkg/metrics"
	"google.golang.org/grpc"
)

func main() {
	var (
		socketPath  string
		nodeID      string
		metricsAddr string
	)
	flag.StringVar(&socketPath, "endpoint", "unix:///tmp/kube-pfs-csi-node.sock", "CSI endpoint")
	flag.StringVar(&nodeID, "node-id", "kube-pfs-node", "node ID")
	flag.StringVar(&metricsAddr, "metrics-listen", ":9104", "metrics listen address")
	flag.Parse()

	_ = metrics.StartServer(metricsAddr)
	log.Printf("csi-node metrics listening on %s", metricsAddr)

	lis, err := listenUnix(socketPath)
	if err != nil {
		log.Fatalf("listen csi node: %v", err)
	}
	defer lis.Close()

	svc := kcsisvc.NewService(nodeID)
	server := grpc.NewServer()
	csipb.RegisterIdentityServer(server, svc)
	csipb.RegisterNodeServer(server, svc)

	log.Printf("csi-node listening on %s", socketPath)
	if err := server.Serve(lis); err != nil {
		log.Fatalf("serve csi-node: %v", err)
	}
}

func listenUnix(endpoint string) (net.Listener, error) {
	const prefix = "unix://"
	if !strings.HasPrefix(endpoint, prefix) {
		return nil, fmt.Errorf("endpoint must start with %q", prefix)
	}
	path := filepath.Clean(endpoint[len(prefix):])
	_ = os.Remove(path)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	return net.Listen("unix", path)
}
