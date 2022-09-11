package driver

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/digitalocean/godo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/util/wait"
)

func (d *Driver) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	fmt.Println("CreateVolume of the controller service was called")

	// name is present
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "CreateVolume must be called with a req name")
	}

	// extract required memory
	// make sure the value here is not less than or = 0
	// requriedBytes is not more than limiteBytes
	sizeBytes := req.CapacityRange.GetRequiredBytes()
	fmt.Println(sizeBytes)
	// make sure volume capabilities have been specified
	if req.VolumeCapabilities == nil || len(req.VolumeCapabilities) == 0 {
		return nil, status.Error(codes.InvalidArgument, "VolumeCapabilities have not been specified")
	}
	// validate volume capabilities
	// make sure accessMode that has been specified by the PVC is actually supported by SP
	// make sure volumeMode that has been specified in the PVC is supported by us

	const gb = 1024 * 1024 * 1024
	// create the request struct
	volReq := godo.VolumeCreateRequest{
		Name:          req.Name,
		Region:        d.region,
		SizeGigaBytes: sizeBytes / gb,
	}
	fmt.Println(volReq)

	// check if volumeContentSource is specified
	// if snapshot is specified, in that case, set the snapshot ID in the volume reqeust
	// you will also have to make sure that this snapshot is actually present
	// volReq.SnapshotID = req.VolumeContentSource.GetSnapshot().SnapshotId

	// if this user have not exceeded the limit
	// if this user can provision the requested amount etc

	// handle AccessibilityRequirements

	// actually call DO api to create the volume
	vol, _, err := d.storage.CreateVolume(ctx, &volReq)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed provisoing the volume error %s\n", err.Error()))
	}

	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			CapacityBytes: sizeBytes,
			VolumeId:      vol.ID,
			// specify content source, but only in cases where its specified in the PVC
		},
	}, nil
}
func (d *Driver) DeleteVolume(context.Context, *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	return nil, nil
}
func (d *Driver) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	fmt.Println("ControllerPublishVolume of controller plugin was called")

	// check if volumeID is present and volume is available on SP
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "VolumeID is mandatory in ControllerPublishVolume request")
	}

	// if nodeID is set, and node is actually present on SP
	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeID is mandatory in CPVolume request")
	}

	vol, _, err := d.storage.GetVolume(ctx, req.VolumeId)
	if err != nil {
		return nil, status.Error(codes.Internal, "Volume is not available anymore")
	}

	// check if the volume is already attache
	// if it's attached to the correct node
	// or its attached to a wrong node

	// check readOnly is set, and you support readonly volumes
	// also check volumeCaps

	nodeID, err := strconv.Atoi(req.NodeId)
	if err != nil {
		return nil, status.Error(codes.Internal, "was not able to convert nodeID to int value")
	}
	action, _, err := d.storageAction.Attach(ctx, req.VolumeId, nodeID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed attaching volume to the node, error %s", err.Error()))
	}

	if err := d.waitForCompletion(req.VolumeId, action.ID); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("error %s, waiting for volume to be attached", err.Error()))
	}

	return &csi.ControllerPublishVolumeResponse{
		PublishContext: map[string]string{
			volNameKeyFromContPub: vol.Name,
		},
	}, nil
}

func (d *Driver) waitForCompletion(volID string, actionID int) error {
	err := wait.Poll(1*time.Second, 5*time.Minute, func() (done bool, err error) {
		a, _, err := d.storageAction.Get(context.Background(), volID, actionID)
		if err != nil {
			return false, nil
		}

		if a.Status == godo.ActionCompleted {
			return true, nil
		}
		return false, nil
	})
	return err
}

func (d *Driver) ControllerUnpublishVolume(context.Context, *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return nil, nil
}
func (d *Driver) ValidateVolumeCapabilities(context.Context, *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	return nil, nil
}
func (d *Driver) ListVolumes(context.Context, *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	return nil, nil
}
func (d *Driver) GetCapacity(context.Context, *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, nil
}
func (d *Driver) ControllerGetCapabilities(context.Context, *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	caps := []*csi.ControllerServiceCapability{}

	for _, c := range []csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
	} {
		caps = append(caps, &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: c,
				},
			},
		})
	}

	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: caps,
	}, nil
}
func (d *Driver) CreateSnapshot(context.Context, *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, nil
}
func (d *Driver) DeleteSnapshot(context.Context, *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, nil
}
func (d *Driver) ListSnapshots(context.Context, *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, nil
}
func (d *Driver) ControllerExpandVolume(context.Context, *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	return nil, nil
}
func (d *Driver) ControllerGetVolume(context.Context, *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	return nil, nil
}
