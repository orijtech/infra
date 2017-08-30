# infra [![Godoc](https://godoc.org/github.com/orijtech/infra?status.svg)](https://godoc.org/github.com/orijtech/infra)
Cloud server infrastructure that is used to easily coordinate with Google Cloud Platform
with helpers for interacting with:

- Google Cloud DNS
- Google Compute Engine
- Google Cloud Storage

With applications such as:
- Setting up frontend servers fully connected to Google Cloud DNS, and proxying traffic
to backends for example, if the program below is run, it'll add Google Cloud DNS entries
automatically, making the respective CNAMES, A records etc, and thus visiting the domain
name or aliases will resolve and send traffic to the provided IPV4 addresses.
```go
package main

import (
	"fmt"
	"log"

	"github.com/orijtech/infra"
)

func main() {
	infraClient, err := infra.NewDefaultClient()
	if err != nil {
		log.Fatal(err)
	}

	setupInfo, err := infraClient.FullSetup(&infra.Setup{
		Project: "sample-961732",
		Zone:    "us-central1-c",

		ProjectDescription: "full-setup",
		MachineName:        "full-setup-sample",

		DomainName:   "edison.orijtech.com",
		ProxyAddress: "http://10.128.0.5/",
		Aliases:      []string{"www.edison.orijtech.com", "el.orijtech.com"},

		IPV4Addresses: []string{"37.162.3.87"},
	})

	if err != nil {
		log.Fatal(err)
	}
	log.Printf("SetupResponse: %#v\n", setupInfo)
}
```

- Creating a Google Compute Engine instance
```go
package main

import (
	"fmt"
	"log"
	"encoding/json"

	"github.com/orijtech/infra"
)

func main() {
	client, err := infra.NewDefaultClient()
	if err != nil {
		log.Fatal(err)
	}
	instance, err := client.CreateInstance(&infra.InstanceRequest{
		Description: "Git server",

		Project: "sample-998172",
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
```

- Add record sets to Google Cloud DNS
```go
package main

import (
	"fmt"
	"log"

	"github.com/orijtech/infra"
)

func main() {
	client, err := infra.NewDefaultClient()
	if err != nil {
		log.Fatal(err)
	}
	addRes, err := client.AddRecordSets(&infra.UpdateRequest{
		Project: "sample-981058",
		Zone:    "us-central1-c",

		Records: []*infra.Record{
			{
				Type: infra.AName, DNSName: "git.orijtech.com.",
				IPV4Addresses: []string{"108.11.144.83"},
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
```

- Uploading to Google Cloud Storage
```go
package main

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/orijtech/infra"
)

func main() {
	infraClient, err := infra.NewDefaultClient()
	if err != nil {
		log.Fatal(err)
	}

	outParams := &infra.UploadParams{
		Reader: func() io.Reader { return strings.NewReader("This is an upload") },
		Name:   "foo",
		Bucket: "bucket",
		Public: true,
	}

	obj, err := infraClient.UploadWithParams(outParams)
	if err != nil {
		log.Fatalf("uploadWithParams: %v", err)
	}
	fmt.Printf("The URL: %s\n", infra.ObjectURL(obj))
	fmt.Printf("Size: %d\n", obj.Size)
}
```

- For more examples, see file [example_test](./example_test.go)
