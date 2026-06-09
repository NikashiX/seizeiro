package blob

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
)

var _ Storage = (*AzureStorage)(nil)

// AzureStorage é uma implementação de [Storage] que armazena os objetos como blobs
// em um container do Azure Blob Storage.
type AzureStorage struct {
	client    *azblob.Client
	container string
}

// NewAzureStorage cria um [AzureStorage] que opera sobre o container informado na
// conta de armazenamento account.
//
// A URL do serviço é montada como "https://<account>.blob.core.windows.net" e a
// autenticação usa [azidentity.NewDefaultAzureCredential].
func NewAzureStorage(account, container string) (*AzureStorage, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("default credential: %w", err)
	}

	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net", account)
	client, err := azblob.NewClient(serviceURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("new client: %w", err)
	}

	return &AzureStorage{
		client:    client,
		container: container,
	}, nil
}

// Get baixa o blob identificado por key. Retorna [ErrNotFound] caso não exista.
func (s *AzureStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	res, err := s.client.DownloadStream(ctx, s.container, key, nil)
	if bloberror.HasCode(err, bloberror.BlobNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("download stream: %w", err)
	}
	return res.Body, nil
}

// Put envia o conteúdo lido de r para o blob identificado por key, sobrescrevendo
// o blob caso já exista. O contentType é gravado como o header Content-Type do blob.
func (s *AzureStorage) Put(ctx context.Context, key, contentType string, r io.Reader) error {
	opts := &azblob.UploadStreamOptions{
		HTTPHeaders: &blob.HTTPHeaders{
			BlobContentType: &contentType,
		},
	}
	_, err := s.client.UploadStream(ctx, s.container, key, r, opts)
	if err != nil {
		return fmt.Errorf("upload stream: %w", err)
	}
	return nil
}

// Delete remove o blob identificado por key. É idempotente: caso o blob não exista,
// retorna nil.
func (s *AzureStorage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteBlob(ctx, s.container, key, nil)
	if err != nil && !bloberror.HasCode(err, bloberror.BlobNotFound) {
		return fmt.Errorf("delete blob: %w", err)
	}
	return nil
}

// ContainerPermissions descreve as permissões concedidas por uma URL SAS de container.
type ContainerPermissions struct {
	Read   bool
	Write  bool
	List   bool
	Delete bool
}

// ContainerSAS gera uma URL SAS de delegação de usuário para o container, incluindo o
// token SAS na query string, válida por ttl e com as permissões informadas.
func (s *AzureStorage) ContainerSAS(ctx context.Context, perms ContainerPermissions, ttl time.Duration) (string, error) {
	svc := s.client.ServiceClient()

	// Recua o início em alguns segundos para tolerar diferenças de relógio.
	start := time.Now().UTC().Add(-10 * time.Second)
	expiry := start.Add(ttl)

	info := service.KeyInfo{
		Start:  new(start.Format(sas.TimeFormat)),
		Expiry: new(expiry.Format(sas.TimeFormat)),
	}
	udc, err := svc.GetUserDelegationCredential(ctx, info, nil)
	if err != nil {
		return "", fmt.Errorf("user delegation credential: %w", err)
	}

	azPerms := sas.ContainerPermissions{
		Read:   perms.Read,
		Write:  perms.Write,
		List:   perms.List,
		Delete: perms.Delete,
	}
	qp, err := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPS,
		StartTime:     start,
		ExpiryTime:    expiry,
		Permissions:   azPerms.String(),
		ContainerName: s.container,
	}.SignWithUserDelegation(udc)

	if err != nil {
		return "", fmt.Errorf("sign with user delegation: %w", err)
	}

	containerURL := strings.TrimSuffix(svc.URL(), "/") + "/" + s.container
	return containerURL + "?" + qp.Encode(), nil
}
