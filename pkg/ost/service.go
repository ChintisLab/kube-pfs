package ost

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rachanaanugandula/kube-pfs/pkg/metrics"
	protogen "github.com/rachanaanugandula/kube-pfs/pkg/proto/gen"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	protogen.UnimplementedObjectStorageServiceServer

	ostID      string
	dataDir    string
	iopsTotal  atomic.Uint64
	bytesTotal atomic.Uint64
	latencyNS  atomic.Uint64
}

func NewService(ostID, dataDir string) (*Service, error) {
	if ostID == "" {
		return nil, errors.New("ost id is required")
	}
	if dataDir == "" {
		return nil, errors.New("data dir is required")
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	return &Service{ostID: ostID, dataDir: dataDir}, nil
}

func (s *Service) WriteBlock(_ context.Context, req *protogen.WriteBlockRequest) (*protogen.WriteBlockResponse, error) {
	start := time.Now()
	defer s.observe("write", len(req.GetData()), start)

	if req.GetBlock() == nil {
		return nil, status.Error(codes.InvalidArgument, "block is required")
	}
	path := s.blockPath(req.GetBlock())
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, status.Errorf(codes.Internal, "mkdir block parent: %v", err)
	}
	if err := os.WriteFile(path, req.GetData(), 0644); err != nil {
		return nil, status.Errorf(codes.Internal, "write block: %v", err)
	}
	return &protogen.WriteBlockResponse{BytesWritten: uint64(len(req.GetData()))}, nil
}

func (s *Service) ReadBlock(_ context.Context, req *protogen.ReadBlockRequest) (*protogen.ReadBlockResponse, error) {
	start := time.Now()
	defer s.observe("read", 0, start)

	if req.GetBlock() == nil {
		return nil, status.Error(codes.InvalidArgument, "block is required")
	}
	path := s.blockPath(req.GetBlock())
	blob, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, status.Error(codes.NotFound, "block not found")
		}
		return nil, status.Errorf(codes.Internal, "read block: %v", err)
	}
	offset := req.GetOffset()
	if offset > uint64(len(blob)) {
		return &protogen.ReadBlockResponse{Data: []byte{}}, nil
	}
	blob = blob[offset:]
	if req.GetLength() > 0 && req.GetLength() < uint64(len(blob)) {
		blob = blob[:req.GetLength()]
	}
	metrics.AddReadThroughput("ost", s.ostID, len(blob))
	return &protogen.ReadBlockResponse{Data: blob}, nil
}

func (s *Service) DeleteBlock(_ context.Context, req *protogen.DeleteBlockRequest) (*protogen.DeleteBlockResponse, error) {
	start := time.Now()
	defer s.observe("delete", 0, start)

	if req.GetBlock() == nil {
		return nil, status.Error(codes.InvalidArgument, "block is required")
	}
	path := s.blockPath(req.GetBlock())
	err := os.Remove(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &protogen.DeleteBlockResponse{Deleted: false}, nil
		}
		return nil, status.Errorf(codes.Internal, "delete block: %v", err)
	}
	return &protogen.DeleteBlockResponse{Deleted: true}, nil
}

func (s *Service) GetHealth(context.Context, *protogen.HealthRequest) (*protogen.HealthResponse, error) {
	return &protogen.HealthResponse{
		OstId:           s.ostID,
		Healthy:         true,
		IopsTotal:       s.iopsTotal.Load(),
		ThroughputBytes: s.bytesTotal.Load(),
	}, nil
}

func (s *Service) blockPath(ref *protogen.BlockRef) string {
	fileID := sanitize(ref.GetFileId())
	chunkID := fmt.Sprintf("%d", ref.GetChunkId())
	return filepath.Join(s.dataDir, fileID, chunkID+".blk")
}

func (s *Service) observe(op string, bytes int, started time.Time) {
	s.iopsTotal.Add(1)
	metrics.IncIOPS("ost", s.ostID, op)
	if op == "write" {
		metrics.ObserveWriteLatency("ost", s.ostID, time.Since(started))
	}
	if bytes > 0 {
		s.bytesTotal.Add(uint64(bytes))
	}
	s.latencyNS.Add(uint64(time.Since(started).Nanoseconds()))
}

func sanitize(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "..", "_")
	if s == "" {
		return "unknown"
	}
	return s
}
