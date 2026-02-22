package main

import (
	"flag"
	"log"
	"net"

	"github.com/rachanaanugandula/kube-pfs/pkg/ost"
	protogen "github.com/rachanaanugandula/kube-pfs/pkg/proto/gen"
	"google.golang.org/grpc"
)

func main() {
	var (
		listenAddr = flag.String("listen", ":50061", "gRPC listen address")
		ostID      = flag.String("ost-id", "ost-0", "OST node ID")
		dataDir    = flag.String("data-dir", "./data/ost", "OST data directory")
	)
	flag.Parse()

	svc, err := ost.NewService(*ostID, *dataDir)
	if err != nil {
		log.Fatalf("init ost service: %v", err)
	}

	lis, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatalf("listen ost: %v", err)
	}

	grpcServer := grpc.NewServer()
	protogen.RegisterObjectStorageServiceServer(grpcServer, svc)

	log.Printf("ost(%s) listening on %s", *ostID, *listenAddr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("serve ost: %v", err)
	}
}
