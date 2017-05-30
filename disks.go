package infra

import (
	"google.golang.org/api/compute/v1"
)

var (
	BasicAttachedDisk = &compute.AttachedDisk{
		AutoDelete: true,

		Boot: true,
		Type: "PERSISTENT",
		Mode: "READ_WRITE",

		InitializeParams: &compute.AttachedDiskInitializeParams{
			DiskSizeGb:  10,
			SourceImage: "projects/debian-cloud/global/images/family/debian-8",
		},
	}
)
