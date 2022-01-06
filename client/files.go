package chclient

import (
	"context"

	"github.com/cloudradar-monitoring/rport/share/files"

	"github.com/cloudradar-monitoring/rport/share/models"
)

func (c *Client) HandleUploadRequest(ctx context.Context, reqPayload []byte) error {
	uploadedFile := new(models.UploadedFile)
	err := uploadedFile.FromMultipartData(reqPayload)
	if err != nil {
		return err
	}

	err = uploadedFile.Validate()
	if err != nil {
		return err
	}

	fileExists, err := files.FileOrDirExists(uploadedFile.DestinationPath)
	if err != nil {
		return err
	}

	if fileExists && !uploadedFile.ForceWrite {
		c.Logger.Infof("file %s already exists and not forced to be overwritten, will skip the request", uploadedFile.DestinationPath)
		return nil
	}

	err = files.CreateFile(c.configHolder.GetUploadDir(), uploadedFile, c.Logger)
	if err != nil {
		return err
	}

	err = files.MoveFileToDestination(uploadedFile, c.Logger)
	if err != nil {
		return err
	}

	err = files.ChangeOwner(uploadedFile, c.Logger)
	if err != nil {
		return err
	}

	return nil
}
