package infra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2/google"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/dns/v1"
	"google.golang.org/api/storage/v1"
)

var defaultGCEScopes = []string{}

type Client struct {
	computeSrvc *compute.Service
	dnsSrvc     *dns.Service
	storageSrvc *storage.Service
}

func NewWithHTTPClient(hc *http.Client) (*Client, error) {
	computeSrvc, err := compute.New(hc)
	if err != nil {
		return nil, err
	}
	dnsSrvc, err := dns.New(hc)
	if err != nil {
		return nil, err
	}
	storageSrvc, err := storage.New(hc)
	if err != nil {
		return nil, err
	}

	c := &Client{
		computeSrvc: computeSrvc,
		dnsSrvc:     dnsSrvc,
		storageSrvc: storageSrvc,
	}
	return c, nil
}

func NewDefaultClient(scopes ...string) (*Client, error) {
	if len(scopes) == 0 {
		scopes = defaultGCEScopes[:]
	}
	httpClient, err := google.DefaultClient(context.Background(), scopes...)
	if err != nil {
		return nil, err
	}
	return NewWithHTTPClient(httpClient)
}

func (c *Client) zonesService() *compute.ZonesService {
	return compute.NewZonesService(c.computeSrvc)
}

func (c *Client) instancesService() *compute.InstancesService {
	return compute.NewInstancesService(c.computeSrvc)
}

type ZonePage struct {
	Err        error
	PageNumber int64           `json:"page_number"`
	Zones      []*compute.Zone `json:"zones,omitempty"`
}

type ZoneRequest struct {
	Project string `json:"project"`

	OrderBy string `json:"order_by"`
	Filter  string `json:"filter"`

	MaxPages       int64 `json:"max_pages"`
	ResultsPerPage int64 `json:"results_per_page"`
}

type ZonePagesResponse struct {
	Pages  <-chan *ZonePage
	Cancel func() error
}

type InstancePage struct {
	Err        error
	PageNumber int64               `json:"page_number"`
	Instances  []*compute.Instance `json:"instances,omitempty"`
}

type InstancePagesResponse struct {
	Pages  <-chan *InstancePage
	Cancel func() error
}

func makeCanceler() (<-chan bool, func() error) {
	var cancelOnce sync.Once
	cancelChan := make(chan bool, 1)
	cancel := func() error {
		var err error
		cancelOnce.Do(func() {
			close(cancelChan)
		})
		return err
	}

	return cancelChan, cancel
}

var (
	errBlankProject    = errors.New("expecting a non-blank project")
	errBlankZone       = errors.New("expecting a non-blank zone")
	errEmptyInstanceID = errors.New("expecting a non-empty instanceID")
	errEmptyProject    = errors.New("expecting a non-empty project")
	errEmptyZone       = errors.New("expecting a non-empty zone")
	errBlankName       = errors.New("expecting a non-blank name")
	errUnimplemented   = errors.New("unimplemented")

	errEmptyNetworkInterface = errors.New("expecting a non-blank network interface")
)

func (zreq *ZoneRequest) Validate() error {
	if zreq == nil || zreq.Project == "" {
		return errBlankProject
	}
	return nil
}

type InstancesRequest struct {
	Project string `json:"project"`

	OrderBy string `json:"order_by"`
	Filter  string `json:"filter"`

	MaxPages       int64 `json:"max_pages"`
	ResultsPerPage int64 `json:"results_per_page"`

	Zone string `json:"zone"`
}

func (ireq *InstancesRequest) Validate() error {
	if ireq == nil || ireq.Zone == "" {
		return errBlankZone
	}
	if ireq.Project == "" {
		return errBlankProject
	}
	return nil
}

func (c *Client) ListInstances(req *InstancesRequest) (*InstancePagesResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	maxPageNumber := req.MaxPages
	pageExceedsMax := func(pageNumber int64) bool {
		if maxPageNumber <= 0 {
			return false
		}
		return pageNumber > maxPageNumber
	}

	maxResultsPerPage := int64(40)
	if req.ResultsPerPage > 0 {
		maxResultsPerPage = req.ResultsPerPage
	}

	cancelChan, cancelFn := makeCanceler()
	pagesChan := make(chan *InstancePage)
	go func() {
		defer close(pagesChan)

		ilc := c.instancesService().List(req.Project, req.Zone)
		ilc.MaxResults(maxResultsPerPage)
		if req.Filter != "" {
			ilc.Filter(req.Filter)
		}

		if req.OrderBy != "" {
			ilc.OrderBy(req.OrderBy)
		}

		pageToken := ""
		pageNumber := int64(0)
		throttleDuration := time.Duration(350 * time.Millisecond)

		for {
			ilc.PageToken(pageToken)
			ipage := new(InstancePage)
			ipage.PageNumber = pageNumber

			ilr, err := ilc.Do()
			if err != nil {
				ipage.Err = err
				pagesChan <- ipage
				return
			}

			ipage.Instances = ilr.Items
			pagesChan <- ipage

			pageNumber += 1
			if pageExceedsMax(pageNumber) {
				return
			}

			pageToken := ilr.NextPageToken

			select {
			case <-cancelChan:
				return
			case <-time.After(throttleDuration):
			}

			if pageToken == "" {
				// No more results left
				break
			}
		}
	}()

	ires := &InstancePagesResponse{
		Pages:  pagesChan,
		Cancel: cancelFn,
	}

	return ires, nil
}

func (c *Client) ListZones(req *ZoneRequest) (*ZonePagesResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	maxPageNumber := req.MaxPages
	pageExceedsMax := func(pageNumber int64) bool {
		if maxPageNumber <= 0 {
			return false
		}
		return pageNumber > maxPageNumber
	}

	maxResultsPerPage := int64(40)
	if req.ResultsPerPage > 0 {
		maxResultsPerPage = req.ResultsPerPage
	}

	cancelChan, cancelFn := makeCanceler()
	pagesChan := make(chan *ZonePage)
	go func() {
		defer close(pagesChan)

		zlc := c.zonesService().List(req.Project)
		zlc.MaxResults(maxResultsPerPage)
		if req.Filter != "" {
			zlc.Filter(req.Filter)
		}

		if req.OrderBy != "" {
			zlc.OrderBy(req.OrderBy)
		}

		pageToken := ""
		pageNumber := int64(0)
		throttleDuration := time.Duration(350 * time.Millisecond)

		for {
			zlc.PageToken(pageToken)
			zpage := new(ZonePage)
			zpage.PageNumber = pageNumber

			zlr, err := zlc.Do()
			if err != nil {
				zpage.Err = err
				pagesChan <- zpage
				return
			}

			zpage.Zones = zlr.Items
			pagesChan <- zpage

			pageNumber += 1
			if pageExceedsMax(pageNumber) {
				return
			}

			pageToken := zlr.NextPageToken

			select {
			case <-cancelChan:
				return
			case <-time.After(throttleDuration):
			}

			if pageToken == "" {
				// No more results left
				break
			}
		}
	}()

	zres := &ZonePagesResponse{
		Pages:  pagesChan,
		Cancel: cancelFn,
	}

	return zres, nil
}

type InstanceRequest struct {
	Zone        string       `json:"zone,omitempty"`
	Project     string       `json:"project,omitempty"`
	Description string       `json:"description,omitempty"`
	Name        string       `json:"name,omitempty"`
	MachineType *MachineType `json:"machine_type,omitempty"`

	// CanForwardIP allows an instance to send and receive packets with
	// non-matching destination or source IPs. It is required if you
	// plan to use the instance to forward routes.
	// Description obtained from:
	// https://godoc.org/pkg/google.golang.org/api/compute/v1/#Instance.CanIpForward
	CanForwardIP bool `json:"can_forward_ip,omitempty"`

	Disks []*compute.AttachedDisk `json:"attached_disks,omitempty"`

	// NetworkInterface specifies how this interface is configured to interact with
	// other network services, such as connecting to the internet.
	// Description obtained from:
	// https://godoc.org/google.golang.org/api/compute/v1#Instance.NetworkInterfaces
	NetworkInterface *compute.NetworkInterface `json:"network_interface"`

	// The metadata key/value pairs assigned to the instance.
	// This includes custom metadata and predefined keys.
	// Description obtained from:
	// https://godoc.org/google.golang.org/api/compute/v1#Instance.Metadata
	Metadata *compute.Metadata `json:"metadata"`

	// ServiceAccounts: A list of service accounts, with their specified
	// scopes, authorized for this instance. Only one service account per VM
	// instance is supported.
	//
	// Service accounts generate access tokens that can be accessed through
	// the metadata server and used to authenticate applications on the
	// instance. See Service Accounts for more information.
	// Description obtained from:
	// https://godoc.org/google.golang.org/api/compute/v1#Instance.ServiceAccounts
	ServiceAccounts []*compute.ServiceAccount `json:"service_accounts,omitempty"`

	// NullFields is a list of field names (e.g. "CanIpForward") to include
	// in API requests with the JSON null value. By default, fields with
	// empty values are omitted from API requests. However, any field with
	// an empty field appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in PATCH requests.
	// Description obtained from:
	// https://godoc.org/google.golang.org/api/compute/v1#Instance.NullFields
	NullFields []string `json:"null_fields"`

	// BlockUntilCompletion when set signifies that the instance request
	// should wait until full completion of creation of an instance.
	BlockUntilCompletion bool `json:"block_until_completion"`
}

func (ireq *InstanceRequest) toInstance() *compute.Instance {
	return &compute.Instance{
		Name:  ireq.Name,
		Disks: ireq.disksOrDefault(),

		Metadata:    ireq.Metadata,
		Description: ireq.Description,
		MachineType: ireq.machineTypeOrDefault().partialURLByZone(ireq.Zone),

		ServiceAccounts: ireq.ServiceAccounts[:],

		NetworkInterfaces: []*compute.NetworkInterface{ireq.NetworkInterface},
	}
}

func (ireq *InstanceRequest) disksOrDefault() []*compute.AttachedDisk {
	if len(ireq.Disks) > 0 {
		return ireq.Disks
	}
	return []*compute.AttachedDisk{
		BasicAttachedDisk,
	}
}

func (ireq *InstanceRequest) validateBasic() error {
	if ireq == nil || ireq.Project == "" {
		return errEmptyProject
	}
	if ireq.Zone == "" {
		return errEmptyZone
	}
	if ireq.Name == "" {
		return errBlankName
	}
	return nil
}

func (ireq *InstanceRequest) validateForCreate() error {
	if err := ireq.validateBasic(); err != nil {
		return err
	}
	if ireq.NetworkInterface == nil {
		return errEmptyNetworkInterface
	}
	return ireq.machineTypeOrDefault().Validate()
}

func (ireq *InstanceRequest) machineTypeOrDefault() *MachineType {
	if ireq.MachineType == nil {
		return basic1VCPUMachine
	}
	return ireq.MachineType
}

func (ireq *InstanceRequest) validateForByName() error {
	return ireq.validateBasic()
}

func (c *Client) FindInstance(ireq *InstanceRequest) (*compute.Instance, error) {
	if err := ireq.validateForByName(); err != nil {
		return nil, err
	}
	req := c.instancesService().Get(ireq.Project, ireq.Zone, ireq.Name)
	return req.Do()
}

func (c *Client) CreateInstance(ireq *InstanceRequest) (*compute.Instance, error) {
	if err := ireq.validateForCreate(); err != nil {
		return nil, err
	}
	req := c.instancesService().Insert(ireq.Project, ireq.Zone, ireq.toInstance())
	operation, err := req.Do()
	log.Printf("op: %+v err: %v\n", operation, err)
	if err != nil {
		return nil, err
	}

	// Now check for any errors returned in operations.
	if err := operation.Error; err != nil {
		if anErr, ok := interface{}(err).(error); ok {
			return nil, anErr
		} else {
			jsonBlob, _ := json.Marshal(err)
			return nil, fmt.Errorf("%s", jsonBlob)
		}
	}

	// Then look up the instance by ID since an
	// operation just returns the ID of the item created.
	return c.FindInstance(&InstanceRequest{
		Name:    ireq.Name,
		Zone:    ireq.Zone,
		Project: ireq.Project,

		BlockUntilCompletion: ireq.BlockUntilCompletion,
	})
}
