package infra

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/api/dns/v1"
)

type RecordSetPage struct {
	Err        error
	PageNumber int64 `json:"page_number"`

	RecordSets []*dns.ResourceRecordSet `json:"record_sets,omitempty"`
}

type RecordSetRequest struct {
	Project string `json:"project"`
	Zone    string `json:"zone"`

	// DomainName if set restricts listing to only
	// return records with this fully qualified domain name.
	DomainName string `json:"domain_name"`

	MaxPages       int64 `json:"max_pages"`
	ResultsPerPage int64 `json:"results_per_page"`
}

type RecordSetPagesResponse struct {
	Pages  <-chan *RecordSetPage
	Cancel func() error
}

func (rreq *RecordSetRequest) Validate() error {
	if rreq == nil || rreq.Project == "" {
		return errEmptyProject
	}
	if rreq.Zone == "" {
		return errEmptyZone
	}
	return nil
}

func (c *Client) recordSetsService() *dns.ResourceRecordSetsService {
	return dns.NewResourceRecordSetsService(c.dnsSrvc)
}

func (c *Client) ListDNSRecordSets(rreq *RecordSetRequest) (*RecordSetPagesResponse, error) {
	if err := rreq.Validate(); err != nil {
		return nil, err
	}

	maxPageNumber := rreq.MaxPages
	pageExceedsMax := func(pageNumber int64) bool {
		if maxPageNumber <= 0 {
			return false
		}
		return pageNumber > maxPageNumber
	}

	maxResultsPerPage := int64(40)
	if rreq.ResultsPerPage > 0 {
		maxResultsPerPage = rreq.ResultsPerPage
	}

	cancelChan, cancelFn := makeCanceler()
	pagesChan := make(chan *RecordSetPage)
	go func() {
		defer close(pagesChan)

		dnsLc := c.recordSetsService().List(rreq.Project, rreq.Zone)
		dnsLc.MaxResults(maxResultsPerPage)

		if rreq.DomainName != "" {
			dnsLc.Name(ensureHasTrailingDot(rreq.DomainName))
		}

		pageToken := ""
		pageNumber := int64(0)
		throttleDuration := time.Duration(350 * time.Millisecond)

		for {
			dnsLc.PageToken(pageToken)
			dPage := new(RecordSetPage)
			dPage.PageNumber = pageNumber

			dRes, err := dnsLc.Do()
			if err != nil {
				dPage.Err = err
				pagesChan <- dPage
				return
			}

			dPage.RecordSets = dRes.Rrsets
			pagesChan <- dPage

			pageNumber += 1
			if pageExceedsMax(pageNumber) {
				return
			}

			pageToken := dRes.NextPageToken

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

	rres := &RecordSetPagesResponse{
		Pages:  pagesChan,
		Cancel: cancelFn,
	}

	return rres, nil

}

type RecordType string

const (
	AAAName RecordType = "AAA"
	AName   RecordType = "A"
	CName   RecordType = "CNAME"
	CAA     RecordType = "CAA"
	MX      RecordType = "MX"
	NS      RecordType = "NS"
	SPF     RecordType = "SPF"
	SRV     RecordType = "SRV"
	TXT     RecordType = "TXT"
)

type Record struct {
	DNSName string     `json:"dns_name"`
	TTL     int64      `json:"ttl"`
	Type    RecordType `json:"type"`

	IPV4Addresses []string `json:"ipv4_addresses"`
	IPV6Addresses []string `json:"ipv6_addresses"`
	CanonicalName string   `json:"canonical_name"`

	NameServers []string `json:"name_servers"`

	CertificateAuthorityAuthorizations []string `json:"certificate_authority_authorizations"`

	PreferenceAndMailServers []string `json:"preference_and_mail_servers"`

	SPFData []string `json:"spf_data"`
	SRVData []string `json:"srv_data"`

	TXTRecords []string `json:"txt_records"`
}

func ensureHasTrailingDot(s string) string {
	if !strings.HasSuffix(s, ".") {
		s = s + "."
	}
	return s
}

func (r *Record) toRecordSet() *dns.ResourceRecordSet {
	rrset := &dns.ResourceRecordSet{
		// DNSNames without trailing dots are rejected as
		// invalid so ensure that they do have them.
		Name: ensureHasTrailingDot(r.DNSName),
		Type: string(r.Type),
		Ttl:  r.TTL,
	}

	if r.CanonicalName != "" {
		rrset.Rrdatas = append(rrset.Rrdatas, ensureHasTrailingDot(r.CanonicalName))
	}

	rrset.Rrdatas = append(rrset.Rrdatas, r.CertificateAuthorityAuthorizations...)
	rrset.Rrdatas = append(rrset.Rrdatas, r.IPV4Addresses...)
	rrset.Rrdatas = append(rrset.Rrdatas, r.IPV6Addresses...)
	rrset.Rrdatas = append(rrset.Rrdatas, r.NameServers...)
	rrset.Rrdatas = append(rrset.Rrdatas, r.PreferenceAndMailServers...)
	rrset.Rrdatas = append(rrset.Rrdatas, r.SPFData...)
	rrset.Rrdatas = append(rrset.Rrdatas, r.SRVData...)
	rrset.Rrdatas = append(rrset.Rrdatas, r.TXTRecords...)
	return rrset
}

type UpdateRequest struct {
	Zone    string `json:"zone"`
	Project string `json:"project"`

	Records []*Record `json:"records"`

	Additions []*Record `json:"additions"`
	Deletions []*Record `json:"deletions"`
}

var (
	errBlankCanonicalName = errors.New("expecting a non-blank canonical name")
	errEmptyIPV4Addresses = errors.New("expecting at least one IPV4 address")
	errEmptyIPV6Addresses = errors.New("expecting at least one IPV6 address")

	errEmptyCertificateAuthorityAuthorizations = errors.New("expecting at least one certificate authority authorization")

	errEmptySPFData = errors.New("expecting at least one SPF record")
	errEmptySRVData = errors.New("expecting at least one SRV record")

	errEmptyTXTRecords  = errors.New("expecting at least one TXT record")
	errEmptyNameServers = errors.New("expecting at least one name server")

	errEmptyPreferenceAndMailServers = errors.New("expecting at least one preferenceAndMailServer")

	errBlankUpdateRequest = errors.New("expecting a non-blank updateRequest")
)

func (r *Record) validateForAAAName() error {
	uniqIPV6Addresses := dedup(r.IPV6Addresses...)
	if len(uniqIPV6Addresses) == 0 {
		return errEmptyIPV6Addresses
	}
	r.IPV6Addresses = uniqIPV6Addresses

	return nil
}

func (r *Record) validateForSPF() error {
	uniqs := dedup(r.SPFData...)
	if len(uniqs) == 0 {
		return errEmptySPFData
	}
	r.SPFData = uniqs
	return nil
}

func (r *Record) validateForTXT() error {
	uniqs := dedup(r.TXTRecords...)
	if len(uniqs) == 0 {
		return errEmptyTXTRecords
	}
	r.TXTRecords = uniqs
	return nil
}

func (r *Record) validateForSRV() error {
	uniqs := dedup(r.SRVData...)
	if len(uniqs) == 0 {
		return errEmptySPFData
	}
	r.SRVData = uniqs
	return nil
}

func (r *Record) validateForNS() error {
	uniqs := dedup(r.NameServers...)
	if len(uniqs) == 0 {
		return errEmptyNameServers
	}
	r.NameServers = uniqs
	return nil
}

func (r *Record) validateForMX() error {
	uniqs := dedup(r.PreferenceAndMailServers...)
	if len(uniqs) == 0 {
		return errEmptyPreferenceAndMailServers
	}
	r.PreferenceAndMailServers = uniqs
	return nil
}

func (r *Record) validateForAName() error {
	uniqIPV4Addresses := dedup(r.IPV4Addresses...)
	if len(uniqIPV4Addresses) == 0 {
		return errEmptyIPV4Addresses
	}
	r.IPV4Addresses = uniqIPV4Addresses

	return nil
}

func (r *Record) validateForCName() error {
	if strings.TrimSpace(r.CanonicalName) == "" {
		return errBlankCanonicalName
	}
	return nil
}

func (r *Record) validateForCAA() error {
	uniqs := dedup(r.CertificateAuthorityAuthorizations...)
	if len(uniqs) == 0 {
		return errEmptyCertificateAuthorityAuthorizations
	}
	r.CertificateAuthorityAuthorizations = uniqs
	return nil
}

func dedup(items ...string) []string {
	var uniqRecords []string
	seen := make(map[string]bool)
	for _, item := range items {
		if _, known := seen[item]; known {
			continue
		}
		trimmedItem := strings.TrimSpace(item)
		if trimmedItem == "" {
			continue
		}

		seen[item] = true
		seen[trimmedItem] = true
		uniqRecords = append(uniqRecords, trimmedItem)
	}
	return uniqRecords
}

func (r *Record) Validate() error {
	switch r.Type {
	default:
		return fmt.Errorf("unknown recordType: %q", r.Type)
	case AAAName:
		return r.validateForAAAName()
	case AName:
		return r.validateForAName()
	case CAA:
		return r.validateForCAA()
	case CName:
		return r.validateForCName()
	case MX:
		return r.validateForMX()
	case NS:
		return r.validateForNS()
	case SPF:
		return r.validateForSPF()
	case SRV:
		return r.validateForSRV()
	case TXT:
		return r.validateForTXT()
	}
}

func (ureq *UpdateRequest) validate() error {
	if ureq == nil || ureq.Zone == "" {
		return errBlankZone
	}
	if ureq.Project == "" {
		return errBlankProject
	}
	return nil
}

func (c *Client) UpdateRecordSets(ureq *UpdateRequest) (*dns.Change, error) {
	if err := ureq.validate(); err != nil {
		return nil, err
	}
	deletions, err := toRecordSets(ureq.Deletions...)
	if err != nil {
		return nil, err
	}
	additions, err := toRecordSets(ureq.Additions...)
	if err != nil {
		return nil, err
	}

	change := &dns.Change{
		Additions: additions,
		Deletions: deletions,
	}

	cl := c.changesService().Create(ureq.Project, ureq.Zone, change)
	return cl.Do()
}

func (c *Client) AddRecordSets(areq *UpdateRequest) (*dns.Change, error) {
	if areq == nil {
		return nil, errBlankUpdateRequest
	}

	return c.UpdateRecordSets(&UpdateRequest{
		Zone:      areq.Zone,
		Project:   areq.Project,
		Additions: areq.Records[:],
	})
}

func (c *Client) DeleteRecordSets(dreq *UpdateRequest) (*dns.Change, error) {
	if dreq == nil {
		return nil, errBlankUpdateRequest
	}

	return c.UpdateRecordSets(&UpdateRequest{
		Zone:      dreq.Zone,
		Project:   dreq.Project,
		Deletions: dreq.Records[:],
	})
}

func toRecordSets(records ...*Record) ([]*dns.ResourceRecordSet, error) {
	var rrsets []*dns.ResourceRecordSet
	for _, rec := range records {
		if err := rec.Validate(); err != nil {
			return nil, err
		}
		rrsets = append(rrsets, rec.toRecordSet())
	}
	return rrsets, nil
}

func (c *Client) changesService() *dns.ChangesService {
	return dns.NewChangesService(c.dnsSrvc)
}
