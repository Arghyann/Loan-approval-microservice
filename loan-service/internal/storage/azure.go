package storage

import (
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
)

// GenerateUploadURL creates a temporary 15-minute URL the user can upload to
func GenerateUploadURL(fileName string) (string, error) {
	connStr := os.Getenv("AZURE_STORAGE_CONNECTION_STRING")
	containerName := os.Getenv("AZURE_CONTAINER_NAME")

	client, err := azblob.NewClientFromConnectionString(connStr, nil)
	if err != nil {
		return "", err
	}

	permissions := sas.BlobPermissions{
		Create: true,
		Write:  true,
	}
	expiry := time.Now().Add(15 * time.Minute)

	blobClient := client.ServiceClient().NewContainerClient(containerName).NewBlobClient(fileName)
	
	presignedURL, err := blobClient.GetSASURL(permissions, expiry, nil)
	if err != nil {
		return "", err
	}

	return presignedURL, nil
}
