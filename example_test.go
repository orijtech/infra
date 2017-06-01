package infra_test

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/odeke-em/infra"
)

func Example_client_ListZones() {
	client, err := infra.NewDefaultClient()
	if err != nil {
		log.Fatal(err)
	}
	zres, err := client.ListZones(&infra.ZoneRequest{
		Project: "orijtech-161805",
	})
	if err != nil {
		log.Fatal(err)
	}

	for page := range zres.Pages {
		if err := page.Err; err != nil {
			log.Printf("PageNumber: %#d err: %v", page.PageNumber, err)
			continue
		}
		for i, zone := range page.Zones {
			fmt.Printf("#%d: zone: %#v\n", i, zone)
		}
	}
}

func Example_client_ListInstances() {
	client, err := infra.NewDefaultClient()
	if err != nil {
		log.Fatal(err)
	}
	ires, err := client.ListInstances(&infra.InstancesRequest{
		Project: "orijtech-161805",
		Zone:    "us-central1-c",
	})
	if err != nil {
		log.Fatal(err)
	}

	for page := range ires.Pages {
		if err := page.Err; err != nil {
			log.Printf("PageNumber: %#d err: %v", page.PageNumber, err)
			continue
		}
		for i, instance := range page.Instances {
			fmt.Printf("#%d: ID: %d Name: %q MachineType: %#v CPUPlatform: %v Status: %v Disks: %#v\n",
				i, instance.Id, instance.Name, instance.MachineType, instance.CpuPlatform, instance.Status, instance.Disks)
		}
	}
}

func Example_client_CreateInstance() {
	client, err := infra.NewDefaultClient()
	if err != nil {
		log.Fatal(err)
	}
	instance, err := client.CreateInstance(&infra.InstanceRequest{
		Description: "Git server",

		Project: "orijtech-161805",
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
	client, err := infra.NewDefaultClient()
	if err != nil {
		log.Fatal(err)
	}
	instance, err := client.FindInstance(&infra.InstanceRequest{
		Project: "orijtech-161805",
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
	client, err := infra.NewDefaultClient()
	if err != nil {
		log.Fatal(err)
	}
	ires, err := client.ListDNSRecordSets(&infra.RecordSetRequest{
		Project: "orijtech-161805",
		Zone:    "us-central1-c",

		DomainName: "orijtech.com",
	})
	if err != nil {
		log.Fatalf("%+v", err)
	}

	for page := range ires.Pages {
		if err := page.Err; err != nil {
			log.Printf("PageNumber: %#d err: %v", page.PageNumber, err)
			continue
		}
		for i, rset := range page.RecordSets {
			fmt.Printf("#%d: Name: %q TTL: %d Type: %v Rrdatas: %#v\n",
				i, rset.Name, rset.Ttl, rset.Type, rset.Rrdatas)
		}
	}
}

func Example_client_AddRecordSets() {
	client, err := infra.NewDefaultClient()
	if err != nil {
		log.Fatal(err)
	}
	addRes, err := client.AddRecordSets(&infra.UpdateRequest{
		Project: "orijtech-161805",
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
	client, err := infra.NewDefaultClient()
	if err != nil {
		log.Fatal(err)
	}
	delRes, err := client.DeleteRecordSets(&infra.UpdateRequest{
		Project: "orijtech-161805",
		Zone:    "us-central1-c",

		Records: []*infra.Record{
			{
				Type: infra.AName, DNSName: "flick.orijtech.com.",
				IPV4Addresses: []string{"35.184.3.107"},
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
	infraClient, err := infra.NewDefaultClient()
	if err != nil {
		log.Fatal(err)
	}

	setupInfo, err := infraClient.FullSetup(&infra.Setup{
		Project: "orijtech-161805",
		Zone:    "us-central1-c",

		ProjectDescription: "full-setup",
		MachineName:        "full-setup-sample",

		DomainName:   "edison.orijtech.com",
		ProxyAddress: "http://10.128.0.5/",
		Aliases:      []string{"www.edison.orijtech.com", "el.orijtech.com"},

		IPV4Addresses: []string{"35.184.3.107"},
	})

	if err != nil {
		log.Fatal(err)
	}
	log.Printf("SetupResponse: %#v\n", setupInfo)
}
