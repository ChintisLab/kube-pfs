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
	"google.golang.org/grpc"
)

func main() {
	var socketPath string
	flag.StringVar(&socketPath, "endpoint", "unix:///tmp/kube-pfs-csi-controller.sock", "CSI endpoint")
	flag.Parse()

	lis, err := listenUnix(socketPath)
	if err != nil {
		log.Fatalf("listen csi controller: %v", err)
	}
	defer lis.Close()

	svc := kcsisvc.NewService("controller")
	server := grpc.NewServer()
	csipb.RegisterIdentityServer(server, svc)
	csipb.RegisterControllerServer(server, svc)

	log.Printf("csi-controller listening on %s", socketPath)
	if err := server.Serve(lis); err != nil {
		log.Fatalf("serve csi-controller: %v", err)
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
