package smoke

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/rachanaanugandula/kube-pfs/pkg/mds"
	"github.com/rachanaanugandula/kube-pfs/pkg/ost"
	protogen "github.com/rachanaanugandula/kube-pfs/pkg/proto/gen"
)

func TestDay2MDSAndOSTFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	workDir := t.TempDir()

	mdsSvc, err := mds.NewService(mds.Config{
		BoltPath:        filepath.Join(workDir, "mds.db"),
		OSTIDs:          []string{"ost-0", "ost-1", "ost-2"},
		DefaultMode:     0644,
		DefaultStripeSz: 1024 * 1024,
	})
	if err != nil {
		t.Fatalf("new mds service: %v", err)
	}
	t.Cleanup(func() { _ = mdsSvc.Close() })

	ost0, err := ost.NewService("ost-0", filepath.Join(workDir, "ost-0"))
	if err != nil {
		t.Fatalf("new ost-0: %v", err)
	}
	ost1, err := ost.NewService("ost-1", filepath.Join(workDir, "ost-1"))
	if err != nil {
		t.Fatalf("new ost-1: %v", err)
	}
	ost2, err := ost.NewService("ost-2", filepath.Join(workDir, "ost-2"))
	if err != nil {
		t.Fatalf("new ost-2: %v", err)
	}
	ostByID := map[string]*ost.Service{"ost-0": ost0, "ost-1": ost1, "ost-2": ost2}

	createRes, err := mdsSvc.Create(ctx, &protogen.CreateRequest{ParentInodeId: "root", Name: "sample.bin", IsDir: false, Mode: 0644})
	if err != nil {
		t.Fatalf("create file: %v", err)
	}
	inode := createRes.GetInode()
	if inode == nil {
		t.Fatalf("expected inode in create response")
	}
	if len(inode.GetStripeLayout().GetOstIds()) != 3 {
		t.Fatalf("expected 3 ost ids in stripe layout, got %d", len(inode.GetStripeLayout().GetOstIds()))
	}

	targetOST := inode.GetStripeLayout().GetOstIds()[0]
	ostSvc := ostByID[targetOST]
	if ostSvc == nil {
		t.Fatalf("stripe picked unknown ost id: %s", targetOST)
	}

	payload := []byte("day2-smoke-payload")
	_, err = ostSvc.WriteBlock(ctx, &protogen.WriteBlockRequest{
		Block: &protogen.BlockRef{FileId: inode.GetInodeId(), ChunkId: 0, OstId: targetOST},
		Data:  payload,
	})
	if err != nil {
		t.Fatalf("write block: %v", err)
	}

	readRes, err := ostSvc.ReadBlock(ctx, &protogen.ReadBlockRequest{Block: &protogen.BlockRef{FileId: inode.GetInodeId(), ChunkId: 0, OstId: targetOST}})
	if err != nil {
		t.Fatalf("read block: %v", err)
	}
	if string(readRes.GetData()) != string(payload) {
		t.Fatalf("payload mismatch, got=%q want=%q", string(readRes.GetData()), string(payload))
	}

	_, err = mdsSvc.Unlink(ctx, &protogen.UnlinkRequest{ParentInodeId: "root", Name: "sample.bin"})
	if err != nil {
		t.Fatalf("unlink file: %v", err)
	}

	_, err = mdsSvc.Lookup(ctx, &protogen.LookupRequest{ParentInodeId: "root", Name: "sample.bin"})
	if err == nil {
		t.Fatalf("expected lookup to fail after unlink")
	}
}
