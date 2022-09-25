package driver

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	metadata "github.com/digitalocean/go-metadata"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (d *Driver) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	// make sure all the req fields are present
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "VolumeID must be present in the NodeStageVolumeReq")
	}

	if req.StagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "StagingTargetPath must be present in the NodeSVolReq")
	}

	if req.VolumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "VolumeCaps must be present in the NodeSVolReq")
	}

	switch req.VolumeCapability.AccessType.(type) {
	case *csi.VolumeCapability_Block:
		return &csi.NodeStageVolumeResponse{}, nil
	}

	volumeName := ""
	if val, ok := req.PublishContext[volNameKeyFromContPub]; !ok {
		return nil, status.Error(codes.InvalidArgument, "Volumename is not present in the publish context of request")
	} else {
		volumeName = val
	}

	mnt := req.VolumeCapability.GetMount()
	fsType := "ext4"
	if mnt.FsType != "" {
		fsType = mnt.FsType
	}

	// figure out the source and target
	source := getPathFromVolumeName(volumeName)
	target := req.StagingTargetPath

	// format the volume and create a file syste on it
	// mkfs.fstype -F blockdevice
	// mkfs.ext4 -F /dev/...
	err := formatAndMakeFS(source, fsType)
	if err != nil {
		fmt.Printf("unable to create fs error %s\n", err.Error())
		return nil, status.Error(codes.Internal, fmt.Sprintf("unable to create fs error %s\n", err.Error()))
	}

	err = mount(source, target, fsType, mnt.MountFlags)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error %s, mounting the source %s to taget %s\n", err.Error(), source, target))
	}

	return &csi.NodeStageVolumeResponse{}, nil
}

// mount -t type device dir
func mount(source, target, fsType string, options []string) error {
	mountCmd := "mount"

	if fsType == "" {
		return fmt.Errorf("fstype is not provided")
	}

	mountArgs := []string{}
	err := os.MkdirAll(target, 0777)
	if err != nil {
		return fmt.Errorf("error: %s, creating the target dir\n", err.Error())
	}
	mountArgs = append(mountArgs, "-t", fsType)

	// check of options and then append them at the end of the mount command
	if len(options) > 0 {
		mountArgs = append(mountArgs, "-o", strings.Join(options, ","))
	}

	mountArgs = append(mountArgs, source)
	mountArgs = append(mountArgs, target)

	out, err := exec.Command(mountCmd, mountArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error %s, mounting the source %s to tar %s. Output: %s\n", err.Error(), source, target, out)
	}
	return nil
}

func formatAndMakeFS(source, fsType string) error {
	mkfsCmd := fmt.Sprintf("mkfs.%s", fsType)

	_, err := exec.LookPath(mkfsCmd)
	if err != nil {
		return fmt.Errorf("unable to find the mkfs (%s) utiltiy errors is %s", mkfsCmd, err.Error())
	}

	// actually run mkfs.ext4 -F source
	mkfsArgs := []string{"-F", source}

	out, err := exec.Command(mkfsCmd, mkfsArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("create fs command failed output: %s, and err: %s\n", out, err.Error())
	}
	return nil
}

func getPathFromVolumeName(volName string) string {
	return fmt.Sprintf("/dev/disk/by-id/scsi-0DO_Volume_%s", volName)
}

func (d *Driver) NodeUnstageVolume(context.Context, *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return nil, nil
}
func (d *Driver) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	fmt.Printf("NodePublishVolume was called with source %s and target %s\n", req.StagingTargetPath, req.TargetPath)

	// make sure the requried fields are set and not empty

	options := []string{"bind"}
	if req.Readonly {
		options = append(options, "ro")
	}

	// get req.VolumeCaps and make sure that you handle request for block mode as well
	// here we are just handling request for filesystem mode
	// in case of block mode, the source is going to be the device dir where volume was attached form ControllerPubVolume RPC

	fsType := "ext4"
	if req.VolumeCapability.GetMount().FsType != "" {
		fsType = req.VolumeCapability.GetMount().FsType
	}

	source := req.StagingTargetPath
	target := req.TargetPath

	// we want to run mount -t fstype source target -o bind,ro

	err := mount(source, target, fsType, options)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error %s, mounting the volume from staging dir to target dir", err.Error()))
	}

	return &csi.NodePublishVolumeResponse{}, nil
}
func (d *Driver) NodeUnpublishVolume(context.Context, *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	return nil, nil
}
func (d *Driver) NodeGetVolumeStats(context.Context, *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, nil
}
func (d *Driver) NodeExpandVolume(context.Context, *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, nil
}

func (d *Driver) NodeGetCapabilities(context.Context, *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	fmt.Println("NodeGetCaps was called")
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{
			{
				Type: &csi.NodeServiceCapability_Rpc{
					Rpc: &csi.NodeServiceCapability_RPC{
						Type: csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
					},
				},
			},
		},
	}, nil
}
func (d *Driver) NodeGetInfo(context.Context, *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	mdClient := metadata.NewClient()

	id, err := mdClient.DropletID()
	if err != nil {
		return nil, status.Error(codes.Internal, "Error getting nodeID")
	}

	return &csi.NodeGetInfoResponse{
		NodeId:            strconv.Itoa(id),
		MaxVolumesPerNode: 5,
		AccessibleTopology: &csi.Topology{
			Segments: map[string]string{
				"region": "ams3",
			},
		},
	}, nil
}
