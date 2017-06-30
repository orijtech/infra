package infra

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/dns/v1"

	"github.com/odeke-em/frontender"
	"github.com/odeke-em/go-uuid"
)

// Goals:
// 1. Lookup already created domain with zone
//    a) Zone Name
//    b) ZoneID
//
// 2. If proxy server address is provided, well and good
//    otherwise create a single machine instance:
//	  a) Machine Type, CPU cores
//	  b) Get IPv4 address
//
// 3. Make record sets
//    a) A Record
//	+ DNS Name: --> Name, IPV4 Address
//    b) CNAMES ... --> Map to A Record
//
// 4. Deploy frontender server with:
//	Domains --> Record set DNS names.

type Setup struct {
	Project string `json:"project"`
	Zone    string `json:"zone"`

	ProjectDescription string `json:"project_description"`
	MachineName        string `json:"machine_name"`

	DomainName    string   `json:"domainname"`
	IPV4Addresses []string `json:"ipv4_addresses"`
	Aliases       []string `json:"aliases"`
	ProxyAddress  string   `json:"proxy_address"`

	Environ    []string `json:"environ"`
	TargetGOOS string   `json:"target_goos"`
}

var (
	errEmptyDomainName = errors.New("expecting a non-empty domain name")
)

func (req *Setup) Validate() error {
	if req == nil || strings.TrimSpace(req.Project) == "" {
		return errEmptyProject
	}
	if strings.TrimSpace(req.Zone) == "" {
		return errEmptyZone
	}
	if req.DomainName == "" {
		return errEmptyDomainName
	}
	return nil
}

func (c *Client) generateMachineAndIPV4Addresses(req *Setup) ([]string, error) {
	instance, err := c.generateMachine(req)
	if err != nil {
		return nil, err
	}

	if len(instance.NetworkInterfaces) == 0 {
		// Now fetch them directly
		instance, err = c.FindInstance(&InstanceRequest{
			Project: req.Project,
			Zone:    req.Zone,
			Name:    req.MachineName,
		})
		if err != nil {
			// TODO: Rollback the instance?
		}
	}

	return ipv4AddressesFromInstance(instance), nil
}

func ipv4AddressesFromInstance(instance *compute.Instance) []string {
	var ipv4Addresses []string
	for _, netInterface := range instance.NetworkInterfaces {
		ipv4Addresses = append(ipv4Addresses, netInterface.NetworkIP)
	}
	return ipv4Addresses

}

func (c *Client) generateMachine(req *Setup) (*compute.Instance, error) {
	return c.CreateInstance(&InstanceRequest{
		Description: req.ProjectDescription,

		Project: req.Project,
		Zone:    req.Zone,
		Name:    req.MachineName,

		NetworkInterface: BasicExternalNATNetworkInterface,
	})
}

func (c *Client) generateRecordSets(req *Setup, ipv4Addresses ...string) (*dns.Change, error) {
	ireq := &UpdateRequest{
		Project: req.Project,
		Zone:    req.Zone,

		Records: []*Record{
			{
				Type: AName, DNSName: req.DomainName,
				IPV4Addresses: ipv4Addresses[:],
			},
		},
	}

	for _, alias := range req.Aliases {
		ireq.Records = append(ireq.Records, &Record{
			Type:          CName,
			DNSName:       alias,
			CanonicalName: req.DomainName,
		})
	}

	return c.AddRecordSets(ireq)
}

func stripTrailingDot(s string) string { return strings.TrimSuffix(s, ".") }

func recordSetsToDomainNames(recordSets []*dns.ResourceRecordSet, fn func(string) string) []string {
	var domainNames []string
	for _, rset := range recordSets {
		stripped := stripTrailingDot(rset.Name)
		if fn != nil {
			stripped = fn(stripped)
		}
		domainNames = append(domainNames, stripped)
	}
	return domainNames
}

func httpsify(s string) string {
	if strings.HasPrefix(s, "https://") {
		return s
	}
	s = strings.TrimPrefix(s, "http://")
	return "https://" + s
}

func (c *Client) FullSetup(req *Setup) (*SetupResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	ipv4Addresses := req.IPV4Addresses
	if len(ipv4Addresses) == 0 {
		// Time to generate that server
		var err error
		ipv4Addresses, err = c.generateMachineAndIPV4Addresses(req)
		if err != nil {
			return nil, err
		}
	}

	// Now create that DNS mapping:
	dnsChange, err := c.generateRecordSets(req, ipv4Addresses...)
	if err != nil {
		return nil, err
	}

	// Now convert the DNS change additions to https based domains
	httpsDomains := recordSetsToDomainNames(dnsChange.Additions, httpsify)
	nonHTTPSRedirectURL := httpsify(req.DomainName)

	// Now generate the binary
	rc, err := frontender.GenerateBinary(&frontender.DeployInfo{
		FrontendConfig: &frontender.Request{
			Domains:      httpsDomains,
			ProxyAddress: req.ProxyAddress,

			Environ:    req.Environ[:],
			TargetGOOS: req.TargetGOOS,

			NonHTTPSRedirectURL: nonHTTPSRedirectURL,
		},
	})
	if err != nil {
		return nil, err
	}

	// Now upload the binary
	obj, err := c.UploadWithParams(&UploadParams{
		Project: req.Project,
		Public:  true,
		Bucket:  "frontender-binaries",
		Name:    fmt.Sprintf("generated-binary-%s", uuid.NewRandom()),
		Reader:  func() io.Reader { return rc },
	})
	_ = rc.Close()
	if err != nil {
		return nil, err
	}

	resp := &SetupResponse{
		BinaryURL:    ObjectURL(obj),
		DNSAdditions: dnsChange.Additions,
		Domains:      httpsDomains,

		NonHTTPSRedirectURL: nonHTTPSRedirectURL,
	}

	return resp, nil
}

type SetupResponse struct {
	BinaryURL string   `json:"binary_url"`
	Domains   []string `json:"domains"`

	DNSAdditions []*dns.ResourceRecordSet `json:"dns_additions"`

	NonHTTPSRedirectURL string `json:"non_https_redirect_url"`
}
