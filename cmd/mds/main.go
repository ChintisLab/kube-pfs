package main

import (
	"flag"
	"log"
	"net"
	"os"
	"strings"

	"github.com/rachanaanugandula/kube-pfs/pkg/mds"
	protogen "github.com/rachanaanugandula/kube-pfs/pkg/proto/gen"
	"google.golang.org/grpc"
)

func main() {
	var (
		listenAddr = flag.String("listen", ":50051", "gRPC listen address")
		boltPath   = flag.String("bolt-path", "./data/mds.db", "BoltDB path")
		ostIDsRaw  = flag.String("ost-ids", "ost-0,ost-1,ost-2", "comma-separated OST IDs")
	)
	flag.Parse()

	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	svc, err := mds.NewService(mds.Config{
		BoltPath:        *boltPath,
		OSTIDs:          splitCSV(*ostIDsRaw),
		DefaultMode:     0644,
		DefaultStripeSz: 1024 * 1024,
	})
	if err != nil {
		log.Fatalf("init mds service: %v", err)
	}
	defer svc.Close()

	lis, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatalf("listen mds: %v", err)
	}

	grpcServer := grpc.NewServer()
	protogen.RegisterMetadataServiceServer(grpcServer, svc)

	log.Printf("mds listening on %s", *listenAddr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("serve mds: %v", err)
	}
}

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
