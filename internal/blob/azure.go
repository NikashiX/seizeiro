package blob

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
)

// AzureStorage armazena blobs no Azure Blob Storage usando SharedKeyCredential.
// A credencial é obrigatória para gerar SAS URLs assinadas pelo servidor.
type AzureStorage struct {
	cred      *azblob.SharedKeyCredential
	client    *azblob.Client
	container string
	account   string
}

// NewAzureStorage cria o cliente Azure a partir de account/key/container.
func NewAzureStorage(account, key, container string) (*AzureStorage, error) {
	if account == "" || key == "" || container == "" {
		return nil, fmt.Errorf("blob azure: account/key/container obrigatórios")
	}

	cred, err := azblob.NewSharedKeyCredential(account, key)
	if err != nil {
		return nil, fmt.Errorf("blob azure shared key: %w", err)
	}

	accountURL := fmt.Sprintf("https://%s.blob.core.windows.net", account)
	client, err := azblob.NewClientWithSharedKeyCredential(accountURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("blob azure client: %w", err)
	}

	return &AzureStorage{
		cred:      cred,
		client:    client,
		container: container,
		account:   account,
	}, nil
}

func (s *AzureStorage) Put(ctx context.Context, key string, r io.Reader, contentType string) error {
	_, err := s.client.UploadStream(ctx, s.container, key, r, &blockblob.UploadStreamOptions{
		HTTPHeaders: &blob.HTTPHeaders{
			BlobContentType: to.Ptr(contentType),
		},
	})
	if err != nil {
		return fmt.Errorf("blob azure upload: %w", err)
	}
	return nil
}

func (s *AzureStorage) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	resp, err := s.client.DownloadStream(ctx, s.container, key, nil)
	if err != nil {
		if isAzureNotFound(err) {
			return nil, "", ErrNotFound
		}
		return nil, "", fmt.Errorf("blob azure download: %w", err)
	}

	contentType := "application/octet-stream"
	if resp.ContentType != nil && *resp.ContentType != "" {
		contentType = *resp.ContentType
	}

	return resp.Body, contentType, nil
}

func (s *AzureStorage) Exists(ctx context.Context, key string) (bool, error) {
	blobClient := s.client.ServiceClient().NewContainerClient(s.container).NewBlobClient(key)
	_, err := blobClient.GetProperties(ctx, nil)
	if err == nil {
		return true, nil
	}
	if isAzureNotFound(err) {
		return false, nil
	}
	return false, fmt.Errorf("blob azure get properties: %w", err)
}

// PresignedURL devolve a URL do blob com um token SAS de leitura, válido
// por `ttl`.
func (s *AzureStorage) PresignedURL(ctx context.Context, key string, ttl time.Duration) (string, time.Time, error) {
	if ttl <= 0 {
		ttl = time.Hour
	}
	start := time.Now().UTC().Add(-5 * time.Minute) // tolera clock skew
	expiry := time.Now().UTC().Add(ttl)

	sasQuery, err := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPS,
		StartTime:     start,
		ExpiryTime:    expiry,
		Permissions:   (&sas.BlobPermissions{Read: true}).String(),
		ContainerName: s.container,
		BlobName:      key,
	}.SignWithSharedKey(s.cred)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("blob azure sas sign: %w", err)
	}

	url := fmt.Sprintf(
		"https://%s.blob.core.windows.net/%s/%s?%s",
		s.account,
		s.container,
		key,
		sasQuery.Encode(),
	)
	return url, expiry, nil
}

// isAzureNotFound detecta o erro BlobNotFound do SDK.
func isAzureNotFound(err error) bool {
	return bloberror.HasCode(err, bloberror.BlobNotFound)
}
