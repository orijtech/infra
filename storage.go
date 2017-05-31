package infra

import (
	"errors"
	"fmt"
	"io"

	"google.golang.org/api/storage/v1"
)

type UploadParams struct {
	Project string `json:"project"`
	Public  bool   `json:"public"`
	Bucket  string `json:"bucket"`
	Name    string `json:"path"`

	Reader func() io.Reader `json:"-"`
}

var (
	errBlankReaderFunc = errors.New("expecting a non-blank reader function")

	errEmptyName   = errors.New("expecting a non-empty name")
	errEmptyBucket = errors.New("expecting a non-empty bucket")
)

func (params *UploadParams) Validate() error {
	if params == nil || params.Reader == nil {
		return errBlankReaderFunc
	}
	if params.Name == "" {
		return errEmptyName
	}
	if params.Bucket == "" {
		return errEmptyBucket
	}
	return nil
}

type BucketCheck struct {
	Project string `json:"project"`
	Bucket  string `json:"bucket"`
	Public  bool   `json:"public"`
}

func (c *Client) EnsureBucketExists(bc *BucketCheck) (*storage.Bucket, error) {
	foundBucket, err := c.bucketsService().Get(bc.Bucket).Do()
	if err != nil {
		// TODO: Handle the respective error cases e.g:
		// + failure to authenticate
		// + rate limits exceeded
	}

	if foundBucket != nil {
		return foundBucket, nil
	}

	// Otherwise it is time to create that bucket.
	bIns := c.bucketsService().Insert(bc.Project, &storage.Bucket{Name: bc.Bucket})

	var acl = "private"
	if bc.Public {
		acl = "publicRead"
	}
	bIns = bIns.PredefinedDefaultObjectAcl(acl)
	return bIns.Do()
}

func (c *Client) objectsService() *storage.ObjectsService {
	return storage.NewObjectsService(c.storageSrvc)
}

func (c *Client) bucketsService() *storage.BucketsService {
	return storage.NewBucketsService(c.storageSrvc)
}

func (c *Client) UploadWithParams(params *UploadParams) (*storage.Object, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	bucket, err := c.EnsureBucketExists(&BucketCheck{
		Project: params.Project,
		Bucket:  params.Bucket,
	})
	if err != nil {
		return nil, err
	}

	obj := &storage.Object{
		Name:   params.Name,
		Bucket: bucket.Name,
	}

	oIns := c.objectsService().Insert(params.Bucket, obj)

	var acl = "private"
	if params.Public {
		acl = "publicRead"
	}

	oIns = oIns.PredefinedAcl(acl)
	oIns = oIns.Media(params.Reader())
	return oIns.Do()
}

func ObjectURL(obj *storage.Object) string {
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", obj.Bucket, obj.Name)
}
