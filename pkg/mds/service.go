package mds

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rachanaanugandula/kube-pfs/pkg/metrics"
	protogen "github.com/rachanaanugandula/kube-pfs/pkg/proto/gen"
	"go.etcd.io/bbolt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	gproto "google.golang.org/protobuf/proto"
)

const (
	rootInodeID   = "root"
	bucketInodes  = "inodes"
	bucketDirents = "dirents"
)

type Config struct {
	BoltPath        string
	OSTIDs          []string
	DefaultMode     uint64
	DefaultStripeSz uint32
}

type Service struct {
	protogen.UnimplementedMetadataServiceServer

	mu       sync.RWMutex
	db       *bbolt.DB
	inodes   map[string]*protogen.Inode
	dirents  map[string]map[string]string
	ostIDs   []string
	stripeSz uint32
	rr       uint64
}

func NewService(cfg Config) (*Service, error) {
	if len(cfg.OSTIDs) == 0 {
		cfg.OSTIDs = []string{"ost-0", "ost-1", "ost-2"}
	}
	if cfg.DefaultStripeSz == 0 {
		cfg.DefaultStripeSz = 1024 * 1024
	}
	if cfg.DefaultMode == 0 {
		cfg.DefaultMode = 0644
	}

	db, err := bbolt.Open(cfg.BoltPath, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("open boltdb: %w", err)
	}

	s := &Service{
		db:       db,
		inodes:   map[string]*protogen.Inode{},
		dirents:  map[string]map[string]string{},
		ostIDs:   append([]string{}, cfg.OSTIDs...),
		stripeSz: cfg.DefaultStripeSz,
	}

	if err := s.loadOrInitRoot(cfg.DefaultMode); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

func (s *Service) Close() error {
	return s.db.Close()
}

func (s *Service) loadOrInitRoot(defaultMode uint64) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		inodesB, err := tx.CreateBucketIfNotExists([]byte(bucketInodes))
		if err != nil {
			return err
		}
		direntsB, err := tx.CreateBucketIfNotExists([]byte(bucketDirents))
		if err != nil {
			return err
		}

		if err := inodesB.ForEach(func(k, v []byte) error {
			inode := &protogen.Inode{}
			if err := gproto.Unmarshal(v, inode); err != nil {
				return err
			}
			s.inodes[string(k)] = inode
			return nil
		}); err != nil {
			return err
		}

		if err := direntsB.ForEach(func(k, v []byte) error {
			parts := strings.SplitN(string(k), "\x00", 2)
			if len(parts) != 2 {
				return nil
			}
			parent, name := parts[0], parts[1]
			if _, ok := s.dirents[parent]; !ok {
				s.dirents[parent] = map[string]string{}
			}
			s.dirents[parent][name] = string(v)
			return nil
		}); err != nil {
			return err
		}

		if _, ok := s.inodes[rootInodeID]; ok {
			if _, ok := s.dirents[rootInodeID]; !ok {
				s.dirents[rootInodeID] = map[string]string{}
			}
			return nil
		}

		now := time.Now().Unix()
		root := &protogen.Inode{
			InodeId:       rootInodeID,
			ParentInodeId: "",
			Name:          "/",
			IsDir:         true,
			SizeBytes:     0,
			Mode:          defaultMode,
			CreatedUnix:   now,
			ModifiedUnix:  now,
			StripeLayout:  &protogen.StripeLayout{StripeSizeBytes: s.stripeSz, OstIds: append([]string{}, s.ostIDs...)},
		}
		if err := putInode(inodesB, root); err != nil {
			return err
		}
		s.inodes[rootInodeID] = root
		s.dirents[rootInodeID] = map[string]string{}
		return nil
	})
}

func (s *Service) Create(_ context.Context, req *protogen.CreateRequest) (*protogen.CreateResponse, error) {
	waitStart := time.Now()
	s.mu.Lock()
	metrics.ObserveMDSLockContention(time.Since(waitStart))
	defer s.mu.Unlock()

	if req.GetParentInodeId() == "" || req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "parent_inode_id and name are required")
	}
	parent, ok := s.inodes[req.GetParentInodeId()]
	if !ok {
		return nil, status.Error(codes.NotFound, "parent inode not found")
	}
	if !parent.GetIsDir() {
		return nil, status.Error(codes.FailedPrecondition, "parent inode is not a directory")
	}
	if strings.Contains(req.GetName(), "/") {
		return nil, status.Error(codes.InvalidArgument, "name cannot contain '/'")
	}
	if _, ok := s.dirents[parent.GetInodeId()][req.GetName()]; ok {
		return nil, status.Error(codes.AlreadyExists, "entry already exists")
	}

	now := time.Now().Unix()
	inodeID := fmt.Sprintf("inode-%d", time.Now().UnixNano())
	inode := &protogen.Inode{
		InodeId:       inodeID,
		ParentInodeId: parent.GetInodeId(),
		Name:          req.GetName(),
		IsDir:         req.GetIsDir(),
		Mode:          req.GetMode(),
		CreatedUnix:   now,
		ModifiedUnix:  now,
		StripeLayout:  s.nextStripeLayout(),
	}
	if inode.GetMode() == 0 {
		inode.Mode = 0644
	}
	if inode.GetIsDir() {
		inode.StripeLayout = &protogen.StripeLayout{StripeSizeBytes: s.stripeSz, OstIds: append([]string{}, s.ostIDs...)}
	}

	if _, ok := s.dirents[parent.GetInodeId()]; !ok {
		s.dirents[parent.GetInodeId()] = map[string]string{}
	}
	s.dirents[parent.GetInodeId()][inode.GetName()] = inode.GetInodeId()
	s.inodes[inode.GetInodeId()] = inode
	if inode.GetIsDir() {
		s.dirents[inode.GetInodeId()] = map[string]string{}
	}

	if err := s.persistCreate(inode); err != nil {
		return nil, status.Errorf(codes.Internal, "persist create: %v", err)
	}

	return &protogen.CreateResponse{Inode: cloneInode(inode)}, nil
}

func (s *Service) Lookup(_ context.Context, req *protogen.LookupRequest) (*protogen.LookupResponse, error) {
	waitStart := time.Now()
	s.mu.RLock()
	metrics.ObserveMDSLockContention(time.Since(waitStart))
	defer s.mu.RUnlock()

	entries, ok := s.dirents[req.GetParentInodeId()]
	if !ok {
		return nil, status.Error(codes.NotFound, "parent inode not found")
	}
	inodeID, ok := entries[req.GetName()]
	if !ok {
		return nil, status.Error(codes.NotFound, "entry not found")
	}
	inode := s.inodes[inodeID]
	return &protogen.LookupResponse{Inode: cloneInode(inode)}, nil
}

func (s *Service) Stat(_ context.Context, req *protogen.StatRequest) (*protogen.StatResponse, error) {
	waitStart := time.Now()
	s.mu.RLock()
	metrics.ObserveMDSLockContention(time.Since(waitStart))
	defer s.mu.RUnlock()
	inode, ok := s.inodes[req.GetInodeId()]
	if !ok {
		return nil, status.Error(codes.NotFound, "inode not found")
	}
	return &protogen.StatResponse{Inode: cloneInode(inode)}, nil
}

func (s *Service) ListDir(_ context.Context, req *protogen.ListDirRequest) (*protogen.ListDirResponse, error) {
	waitStart := time.Now()
	s.mu.RLock()
	metrics.ObserveMDSLockContention(time.Since(waitStart))
	defer s.mu.RUnlock()
	children, ok := s.dirents[req.GetInodeId()]
	if !ok {
		return nil, status.Error(codes.NotFound, "directory inode not found")
	}

	names := make([]string, 0, len(children))
	for name := range children {
		names = append(names, name)
	}
	sort.Strings(names)

	entries := make([]*protogen.Inode, 0, len(children))
	for _, name := range names {
		inodeID := children[name]
		inode := s.inodes[inodeID]
		if inode != nil {
			entries = append(entries, cloneInode(inode))
		}
	}
	return &protogen.ListDirResponse{Entries: entries}, nil
}

func (s *Service) Unlink(_ context.Context, req *protogen.UnlinkRequest) (*protogen.UnlinkResponse, error) {
	waitStart := time.Now()
	s.mu.Lock()
	metrics.ObserveMDSLockContention(time.Since(waitStart))
	defer s.mu.Unlock()

	children, ok := s.dirents[req.GetParentInodeId()]
	if !ok {
		return nil, status.Error(codes.NotFound, "parent inode not found")
	}
	inodeID, ok := children[req.GetName()]
	if !ok {
		return &protogen.UnlinkResponse{Deleted: false}, nil
	}
	inode := s.inodes[inodeID]
	if inode == nil {
		return &protogen.UnlinkResponse{Deleted: false}, nil
	}
	if inode.GetIsDir() && len(s.dirents[inode.GetInodeId()]) > 0 {
		return nil, status.Error(codes.FailedPrecondition, "directory is not empty")
	}

	delete(children, req.GetName())
	delete(s.inodes, inode.GetInodeId())
	delete(s.dirents, inode.GetInodeId())

	if err := s.persistUnlink(req.GetParentInodeId(), req.GetName(), inode.GetInodeId()); err != nil {
		return nil, status.Errorf(codes.Internal, "persist unlink: %v", err)
	}

	return &protogen.UnlinkResponse{Deleted: true}, nil
}

func (s *Service) nextStripeLayout() *protogen.StripeLayout {
	if len(s.ostIDs) == 0 {
		return &protogen.StripeLayout{StripeSizeBytes: s.stripeSz}
	}
	start := int(atomic.AddUint64(&s.rr, 1)-1) % len(s.ostIDs)
	ordered := make([]string, 0, len(s.ostIDs))
	for i := 0; i < len(s.ostIDs); i++ {
		ordered = append(ordered, s.ostIDs[(start+i)%len(s.ostIDs)])
	}
	return &protogen.StripeLayout{StripeSizeBytes: s.stripeSz, OstIds: ordered}
}

func (s *Service) persistCreate(inode *protogen.Inode) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		inodesB := tx.Bucket([]byte(bucketInodes))
		direntsB := tx.Bucket([]byte(bucketDirents))
		if inodesB == nil || direntsB == nil {
			return errors.New("metadata buckets are missing")
		}
		if err := putInode(inodesB, inode); err != nil {
			return err
		}
		key := []byte(inode.GetParentInodeId() + "\x00" + inode.GetName())
		if err := direntsB.Put(key, []byte(inode.GetInodeId())); err != nil {
			return err
		}
		return nil
	})
}

func (s *Service) persistUnlink(parentInodeID, name, inodeID string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		inodesB := tx.Bucket([]byte(bucketInodes))
		direntsB := tx.Bucket([]byte(bucketDirents))
		if inodesB == nil || direntsB == nil {
			return errors.New("metadata buckets are missing")
		}
		if err := inodesB.Delete([]byte(inodeID)); err != nil {
			return err
		}
		if err := direntsB.Delete([]byte(parentInodeID + "\x00" + name)); err != nil {
			return err
		}
		return nil
	})
}

func putInode(bucket *bbolt.Bucket, inode *protogen.Inode) error {
	blob, err := gproto.Marshal(inode)
	if err != nil {
		return err
	}
	return bucket.Put([]byte(inode.GetInodeId()), blob)
}

func cloneInode(inode *protogen.Inode) *protogen.Inode {
	if inode == nil {
		return nil
	}
	cloned, _ := gproto.Clone(inode).(*protogen.Inode)
	return cloned
}
