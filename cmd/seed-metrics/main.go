package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

	protogen "github.com/rachanaanugandula/kube-pfs/pkg/proto/gen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	var (
		mdsAddr    = flag.String("mds", "127.0.0.1:50051", "metadata service address")
		ostAddr    = flag.String("ost", "127.0.0.1:50061", "object storage service address")
		iterations = flag.Int("n", 15, "number of synthetic operations")
	)
	flag.Parse()

	if *iterations <= 0 {
		log.Fatalf("-n must be > 0")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	mdsConn, err := grpc.DialContext(ctx, *mdsAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalf("connect mds: %v", err)
	}
	defer mdsConn.Close()

	ostConn, err := grpc.DialContext(ctx, *ostAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalf("connect ost: %v", err)
	}
	defer ostConn.Close()

	mdsClient := protogen.NewMetadataServiceClient(mdsConn)
	ostClient := protogen.NewObjectStorageServiceClient(ostConn)

	randSrc := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < *iterations; i++ {
		name := fmt.Sprintf("seed-%d-%d.bin", time.Now().UnixNano(), i)

		createRes, err := mdsClient.Create(ctx, &protogen.CreateRequest{
			ParentInodeId: "root",
			Name:          name,
			IsDir:         false,
			Mode:          0644,
		})
		if err != nil {
			log.Fatalf("mds create failed at iteration %d: %v", i, err)
		}
		inode := createRes.GetInode()
		if inode == nil {
			log.Fatalf("mds create returned nil inode at iteration %d", i)
		}

		_, _ = mdsClient.Lookup(ctx, &protogen.LookupRequest{ParentInodeId: "root", Name: name})
		_, _ = mdsClient.Stat(ctx, &protogen.StatRequest{InodeId: inode.GetInodeId()})
		_, _ = mdsClient.ListDir(ctx, &protogen.ListDirRequest{InodeId: "root"})

		payload := make([]byte, 8192+randSrc.Intn(8192))
		for j := range payload {
			payload[j] = byte(randSrc.Intn(255))
		}

		_, err = ostClient.WriteBlock(ctx, &protogen.WriteBlockRequest{
			Block: &protogen.BlockRef{FileId: inode.GetInodeId(), ChunkId: 0, OstId: "ost-0"},
			Data:  payload,
		})
		if err != nil {
			log.Fatalf("ost write failed at iteration %d: %v", i, err)
		}

		_, err = ostClient.ReadBlock(ctx, &protogen.ReadBlockRequest{
			Block: &protogen.BlockRef{FileId: inode.GetInodeId(), ChunkId: 0, OstId: "ost-0"},
		})
		if err != nil {
			log.Fatalf("ost read failed at iteration %d: %v", i, err)
		}

		_, _ = ostClient.DeleteBlock(ctx, &protogen.DeleteBlockRequest{Block: &protogen.BlockRef{FileId: inode.GetInodeId(), ChunkId: 0, OstId: "ost-0"}})
		_, _ = mdsClient.Unlink(ctx, &protogen.UnlinkRequest{ParentInodeId: "root", Name: name})
	}

	fmt.Printf("seeded metrics with %d synthetic metadata/data operations\n", *iterations)
}
