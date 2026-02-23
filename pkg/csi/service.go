package csi

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	csipb "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/rachanaanugandula/kube-pfs/pkg/metrics"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	pluginName    = "kube-pfs.csi.dev"
	pluginVersion = "0.2.0"
)

type Service struct {
	csipb.UnimplementedIdentityServer
	csipb.UnimplementedControllerServer
	csipb.UnimplementedNodeServer

	nodeID string

	mu      sync.RWMutex
	volumes map[string]*csipb.Volume
}

func NewService(nodeID string) *Service {
	if nodeID == "" {
		nodeID = "kube-pfs-node"
	}
	return &Service{nodeID: nodeID, volumes: map[string]*csipb.Volume{}}
}

func (s *Service) GetPluginInfo(context.Context, *csipb.GetPluginInfoRequest) (*csipb.GetPluginInfoResponse, error) {
	return &csipb.GetPluginInfoResponse{Name: pluginName, VendorVersion: pluginVersion}, nil
}

func (s *Service) GetPluginCapabilities(context.Context, *csipb.GetPluginCapabilitiesRequest) (*csipb.GetPluginCapabilitiesResponse, error) {
	return &csipb.GetPluginCapabilitiesResponse{Capabilities: []*csipb.PluginCapability{
		{Type: &csipb.PluginCapability_Service_{Service: &csipb.PluginCapability_Service{Type: csipb.PluginCapability_Service_CONTROLLER_SERVICE}}},
	}}, nil
}

func (s *Service) Probe(context.Context, *csipb.ProbeRequest) (*csipb.ProbeResponse, error) {
	return &csipb.ProbeResponse{}, nil
}

func (s *Service) CreateVolume(_ context.Context, req *csipb.CreateVolumeRequest) (*csipb.CreateVolumeResponse, error) {
	name := strings.TrimSpace(req.GetName())
	if name == "" {
		metrics.IncCSIOp("CreateVolume", "error")
		return nil, status.Error(codes.InvalidArgument, "volume name is required")
	}
	volumeID := "kube-pfs-" + sanitizeName(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.volumes[volumeID]; ok {
		metrics.IncCSIOp("CreateVolume", "ok")
		return &csipb.CreateVolumeResponse{Volume: existing}, nil
	}
	capacity := int64(1 << 30)
	if req.GetCapacityRange() != nil && req.GetCapacityRange().GetRequiredBytes() > 0 {
		capacity = req.GetCapacityRange().GetRequiredBytes()
	}
	vol := &csipb.Volume{VolumeId: volumeID, CapacityBytes: capacity, VolumeContext: map[string]string{"driver": pluginName}}
	s.volumes[volumeID] = vol
	metrics.IncCSIOp("CreateVolume", "ok")
	return &csipb.CreateVolumeResponse{Volume: vol}, nil
}

func (s *Service) DeleteVolume(_ context.Context, req *csipb.DeleteVolumeRequest) (*csipb.DeleteVolumeResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.volumes, req.GetVolumeId())
	metrics.IncCSIOp("DeleteVolume", "ok")
	return &csipb.DeleteVolumeResponse{}, nil
}

func (s *Service) ControllerPublishVolume(context.Context, *csipb.ControllerPublishVolumeRequest) (*csipb.ControllerPublishVolumeResponse, error) {
	metrics.IncCSIOp("ControllerPublishVolume", "ok")
	return &csipb.ControllerPublishVolumeResponse{PublishContext: map[string]string{}}, nil
}

func (s *Service) ControllerUnpublishVolume(context.Context, *csipb.ControllerUnpublishVolumeRequest) (*csipb.ControllerUnpublishVolumeResponse, error) {
	metrics.IncCSIOp("ControllerUnpublishVolume", "ok")
	return &csipb.ControllerUnpublishVolumeResponse{}, nil
}

func (s *Service) ValidateVolumeCapabilities(_ context.Context, req *csipb.ValidateVolumeCapabilitiesRequest) (*csipb.ValidateVolumeCapabilitiesResponse, error) {
	if len(req.GetVolumeCapabilities()) == 0 {
		metrics.IncCSIOp("ValidateVolumeCapabilities", "error")
		return nil, status.Error(codes.InvalidArgument, "volume capabilities are required")
	}
	metrics.IncCSIOp("ValidateVolumeCapabilities", "ok")
	return &csipb.ValidateVolumeCapabilitiesResponse{Confirmed: &csipb.ValidateVolumeCapabilitiesResponse_Confirmed{VolumeCapabilities: req.GetVolumeCapabilities()}}, nil
}

func (s *Service) ControllerGetCapabilities(context.Context, *csipb.ControllerGetCapabilitiesRequest) (*csipb.ControllerGetCapabilitiesResponse, error) {
	return &csipb.ControllerGetCapabilitiesResponse{Capabilities: []*csipb.ControllerServiceCapability{
		{Type: &csipb.ControllerServiceCapability_Rpc{Rpc: &csipb.ControllerServiceCapability_RPC{Type: csipb.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME}}},
		{Type: &csipb.ControllerServiceCapability_Rpc{Rpc: &csipb.ControllerServiceCapability_RPC{Type: csipb.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME}}},
	}}, nil
}

func (s *Service) NodeStageVolume(_ context.Context, req *csipb.NodeStageVolumeRequest) (*csipb.NodeStageVolumeResponse, error) {
	staging := req.GetStagingTargetPath()
	if staging == "" {
		metrics.IncCSIOp("NodeStageVolume", "error")
		return nil, status.Error(codes.InvalidArgument, "staging target path is required")
	}
	if err := os.MkdirAll(staging, 0755); err != nil {
		metrics.IncCSIOp("NodeStageVolume", "error")
		return nil, status.Errorf(codes.Internal, "create staging path: %v", err)
	}
	metrics.IncCSIOp("NodeStageVolume", "ok")
	return &csipb.NodeStageVolumeResponse{}, nil
}

func (s *Service) NodeUnstageVolume(_ context.Context, req *csipb.NodeUnstageVolumeRequest) (*csipb.NodeUnstageVolumeResponse, error) {
	if req.GetStagingTargetPath() != "" {
		_ = os.RemoveAll(req.GetStagingTargetPath())
	}
	metrics.IncCSIOp("NodeUnstageVolume", "ok")
	return &csipb.NodeUnstageVolumeResponse{}, nil
}

func (s *Service) NodePublishVolume(_ context.Context, req *csipb.NodePublishVolumeRequest) (*csipb.NodePublishVolumeResponse, error) {
	target := req.GetTargetPath()
	staging := req.GetStagingTargetPath()
	if target == "" || staging == "" {
		metrics.IncCSIOp("NodePublishVolume", "error")
		return nil, status.Error(codes.InvalidArgument, "target and staging paths are required")
	}
	if err := os.MkdirAll(target, 0755); err != nil {
		metrics.IncCSIOp("NodePublishVolume", "error")
		return nil, status.Errorf(codes.Internal, "create target path: %v", err)
	}

	marker := filepath.Join(target, ".kube-pfs-mounted")
	if err := os.WriteFile(marker, []byte("staged-at="+staging+"\n"), 0644); err != nil {
		metrics.IncCSIOp("NodePublishVolume", "error")
		return nil, status.Errorf(codes.Internal, "write mount marker: %v", err)
	}
	metrics.IncCSIOp("NodePublishVolume", "ok")
	return &csipb.NodePublishVolumeResponse{}, nil
}

func (s *Service) NodeUnpublishVolume(_ context.Context, req *csipb.NodeUnpublishVolumeRequest) (*csipb.NodeUnpublishVolumeResponse, error) {
	target := req.GetTargetPath()
	if target != "" {
		_ = os.Remove(filepath.Join(target, ".kube-pfs-mounted"))
		_ = os.RemoveAll(target)
	}
	metrics.IncCSIOp("NodeUnpublishVolume", "ok")
	return &csipb.NodeUnpublishVolumeResponse{}, nil
}

func (s *Service) NodeGetInfo(context.Context, *csipb.NodeGetInfoRequest) (*csipb.NodeGetInfoResponse, error) {
	return &csipb.NodeGetInfoResponse{NodeId: s.nodeID}, nil
}

func (s *Service) NodeGetCapabilities(context.Context, *csipb.NodeGetCapabilitiesRequest) (*csipb.NodeGetCapabilitiesResponse, error) {
	return &csipb.NodeGetCapabilitiesResponse{Capabilities: []*csipb.NodeServiceCapability{
		{Type: &csipb.NodeServiceCapability_Rpc{Rpc: &csipb.NodeServiceCapability_RPC{Type: csipb.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME}}},
	}}, nil
}

func sanitizeName(in string) string {
	in = strings.ToLower(strings.TrimSpace(in))
	in = strings.ReplaceAll(in, " ", "-")
	in = strings.ReplaceAll(in, "/", "-")
	in = strings.ReplaceAll(in, "_", "-")
	if in == "" {
		return "vol"
	}
	return in
}
