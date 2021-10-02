package infra_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/orijtech/infra"
)

func Example_client_ListZones() {
	ctx := context.Background()
	client, err := infra.NewDefaultClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	zres, err := client.ListZones(ctx, &infra.ZoneRequest{
		Project: "sample-981058",
	})
	if err != nil {
		log.Fatal(err)
	}

	for page := range zres.Pages {
		if err := page.Err; err != nil {
			log.Printf("PageNumber: #%d err: %v", page.PageNumber, err)
			continue
		}
		for i, zone := range page.Zones {
			fmt.Printf("#%d: zone: %#v\n", i, zone)
		}
	}
}

func Example_client_ListInstances() {
	ctx := context.Background()
	client, err := infra.NewDefaultClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	ires, err := client.ListInstances(ctx, &infra.InstancesRequest{
		Project: "sample-981058",
		Zone:    "us-central1-c",
	})
	if err != nil {
		log.Fatal(err)
	}

	for page := range ires.Pages {
		if err := page.Err; err != nil {
			log.Printf("PageNumber: #%d err: %v", page.PageNumber, err)
			continue
		}
		for i, instance := range page.Instances {
			fmt.Printf("#%d: ID: %d Name: %q MachineType: %#v CPUPlatform: %v Status: %v Disks: %#v\n",
				i, instance.Id, instance.Name, instance.MachineType, instance.CpuPlatform, instance.Status, instance.Disks)
		}
	}
}

func Example_client_CreateInstance() {
	ctx := context.Background()
	client, err := infra.NewDefaultClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	instance, err := client.CreateInstance(ctx, &infra.InstanceRequest{
		Description: "Git server",

		Project: "sample-981058",
		Zone:    "us-central1-c",
		Name:    "git-server",

		NetworkInterface: infra.BasicExternalNATNetworkInterface,
	})
	if err != nil {
		log.Fatal(err)
	}
	blob, _ := json.MarshalIndent(instance, "", "  ")
	fmt.Printf("Retrieved instance: %s\n", blob)
}

func Example_client_FindInstance() {
	ctx := context.Background()
	client, err := infra.NewDefaultClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	instance, err := client.FindInstance(ctx, &infra.InstanceRequest{
		Project: "sample-981058",
		Zone:    "us-central1-c",
		Name:    "archomp",
	})
	if err != nil {
		log.Fatal(err)
	}
	blob, _ := json.MarshalIndent(instance, "", "  ")
	fmt.Printf("Retrieved instance: %s\n", blob)
}

func Example_client_ListDNSRecordSets() {
	ctx := context.Background()
	client, err := infra.NewDefaultClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	ires, err := client.ListDNSRecordSets(ctx, &infra.RecordSetRequest{
		Project: "sample-981058",
		Zone:    "us-central1-c",

		DomainName: "orijtech.com",
	})
	if err != nil {
		log.Fatalf("%+v", err)
	}

	for page := range ires.Pages {
		if err := page.Err; err != nil {
			log.Printf("PageNumber: #%d err: %v", page.PageNumber, err)
			continue
		}
		for i, rset := range page.RecordSets {
			fmt.Printf("#%d: Name: %q TTL: %d Type: %v Rrdatas: %#v\n",
				i, rset.Name, rset.Ttl, rset.Type, rset.Rrdatas)
		}
	}
}

func Example_client_AddRecordSets() {
	ctx := context.Background()
	client, err := infra.NewDefaultClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	addRes, err := client.AddRecordSets(ctx, &infra.UpdateRequest{
		Project: "sample-981058",
		Zone:    "us-central1-c",

		Records: []*infra.Record{
			{
				Type: infra.AName, DNSName: "git.orijtech.com.",
				IPV4Addresses: []string{"130.211.187.103"},
			},

			{Type: infra.CName, DNSName: "www.git.orijtech.com.", CanonicalName: "git.orijtech.com."},
			{Type: infra.CName, DNSName: "g.orijtech.com.", CanonicalName: "git.orijtech.com."},
		},
	})
	if err != nil {
		log.Fatalf("%+v", err)
	}

	fmt.Printf("addRes: %+v\n", addRes)
}

func Example_client_DeleteRecordSets() {
	ctx := context.Background()
	client, err := infra.NewDefaultClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	delRes, err := client.DeleteRecordSets(ctx, &infra.UpdateRequest{
		Project: "sample-981058",
		Zone:    "us-central1-c",

		Records: []*infra.Record{
			{
				Type: infra.AName, DNSName: "flick.orijtech.com.",
				IPV4Addresses: []string{"37.45.3.107"},
			},
			{Type: infra.CName, DNSName: "el.orijtech.com.", CanonicalName: "edison.orijtech.com."},
			{Type: infra.CName, DNSName: "tset.orijtech.com.", CanonicalName: "edison.orijtech.com."},
			{Type: infra.CName, DNSName: "f.orijtech.com.", CanonicalName: "fullsetup.orijtech.com."},
			{Type: infra.CName, DNSName: "fli.orijtech.com.", CanonicalName: "fullsetup.orijtech.com."},
			{Type: infra.CName, DNSName: "flic.orijtech.com.", CanonicalName: "fullsetup.orijtech.com."},
		},
	})
	if err != nil {
		log.Fatalf("%+v", err)
	}

	fmt.Printf("delRes: %+v\n", delRes)
}

func Example_client_FullSetup() {
	ctx := context.Background()
	infraClient, err := infra.NewDefaultClient(ctx)
	if err != nil {
		log.Fatal(err)
	}

	setupInfo, err := infraClient.FullSetup(ctx, &infra.Setup{
		Project: "sample-981058",
		Zone:    "us-central1-c",

		ProjectDescription: "full-setup",
		MachineName:        "full-setup-sample",

		DomainName:   "edison.orijtech.com",
		ProxyAddress: "http://10.128.0.5/",
		Aliases:      []string{"www.edison.orijtech.com", "el.orijtech.com"},

		IPV4Addresses: []string{"37.45.3.107"},
	})

	if err != nil {
		log.Fatal(err)
	}
	log.Printf("SetupResponse: %#v\n", setupInfo)
}

func Example_client_Upload() {
	ctx := context.Background()
	infraClient, err := infra.NewDefaultClient(ctx)
	if err != nil {
		log.Fatal(err)
	}

	outParams := &infra.UploadParams{
		Reader: func() io.Reader { return strings.NewReader("This is an upload") },
		Name:   "foo",
		Bucket: "bucket",
		Public: true,
	}

	obj, err := infraClient.UploadWithParams(ctx, outParams)
	if err != nil {
		log.Fatalf("uploadWithParams: %v", err)
	}
	fmt.Printf("The URL: %s\n", infra.ObjectURL(obj))
	fmt.Printf("Size: %d\n", obj.Size)
}

func Example_client_Download() {
	ctx := context.Background()
	infraClient, err := infra.NewDefaultClient(ctx)
	if err != nil {
		log.Fatal(err)
	}

	body, err := infraClient.Download(ctx, "archomp", "demos/gears.gif")
	if err != nil {
		log.Fatalf("Download: %v", err)
	}
	defer body.Close()

	f, err := os.Create("gears.gif")
	if err != nil {
		log.Fatalf("Creating bar.png: %v", err)
	}
	defer f.Close()

	n, err := io.Copy(f, body)
	if err != nil {
		log.Fatalf("io.Copy err: %v", err)
	}
	fmt.Printf("Wrote %d bytes to disk!\n", n)
}
