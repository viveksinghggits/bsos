package main

import (
	"flag"
	"fmt"

	"github.com/viveksinghggits/bsos/pkg/driver"
)

func main() {

	var (
		endpoint = flag.String("endpoint", "defaultValue", "Endpoint our gRPC server would run at")
		token    = flag.String("token", "defaultValue", "token of the storage provider")
		region   = flag.String("region", "ams3", "region wher the volumes are going to be provisioned")
	)
	flag.Parse()

	fmt.Println(*endpoint, *token, *region)

	// create a driver instance
	drv, err := driver.NewDriver(driver.InputParams{
		Name: driver.DefaultName,
		// unix:///var/lib/csi/sockets/csi.sock
		Endpoint: *endpoint,
		Region:   *region,
		Token:    *token,
	})
	if err != nil {
		fmt.Printf("Error %s, creating new instance of driver", err.Error())
		return
	}

	// run on that driver instance, it would start the gRPC server
	if err := drv.Run(); err != nil {
		fmt.Printf("Error %s, running the driver", err.Error())
	}
}
