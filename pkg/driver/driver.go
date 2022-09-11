package driver

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/digitalocean/godo"
	"google.golang.org/grpc"
)

const (
	DefaultName = "bsos.viveksingh.dev"
)

var (
	volNameKeyFromContPub = "bsos.viveksingh.dev/volume-name"
)

type Driver struct {
	name     string
	region   string
	endpoint string

	srv *grpc.Server
	// http server, health check
	// storage clients
	storage       godo.StorageService
	storageAction godo.StorageActionsService

	ready bool
}

type InputParams struct {
	Name     string
	Endpoint string
	Token    string
	Region   string
}

func NewDriver(params InputParams) (*Driver, error) {
	if params.Token == "" {
		return nil, errors.New("token must be specified")
	}

	client := godo.NewFromToken(params.Token)

	return &Driver{
		name:          params.Name,
		endpoint:      params.Endpoint,
		region:        params.Region,
		storage:       client.Storage,
		storageAction: client.StorageActions,
	}, nil
}

// start the gRPC server, like its mentioned in the CSI spec
func (d *Driver) Run() error {
	url, err := url.Parse(d.endpoint)
	if err != nil {
		return fmt.Errorf("parsing the endpoint %s\n", err.Error())
	}

	if url.Scheme != "unix" {
		return fmt.Errorf("only supported scheme is unix, but provided %s\n", url.Scheme)
	}

	grpcAddress := path.Join(url.Host, filepath.FromSlash(url.Path))
	if url.Host == "" {
		grpcAddress = filepath.FromSlash(url.Path)
	}

	if err := os.Remove(grpcAddress); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removiong listen address %s\n", err.Error())
	}

	listener, err := net.Listen(url.Scheme, grpcAddress)
	if err != nil {
		return fmt.Errorf(".Listen failed %s\n", err.Error())
	}
	fmt.Println(listener)
	d.srv = grpc.NewServer()

	csi.RegisterNodeServer(d.srv, d)
	csi.RegisterControllerServer(d.srv, d)
	csi.RegisterIdentityServer(d.srv, d)

	d.ready = true

	return d.srv.Serve(listener)
}
