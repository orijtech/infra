package infra

import (
	"google.golang.org/api/compute/v1"
)

var (
	BasicExternalNATNetworkInterface = &compute.NetworkInterface{
		AccessConfigs: []*compute.AccessConfig{
			{
				Name: "External NAT",
				Type: "ONE_TO_ONE_NAT",
			},
		},
	}
)
