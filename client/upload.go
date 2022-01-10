package chclient

import (
	"context"
	"os"
	"path"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/share/files"
	"github.com/cloudradar-monitoring/rport/share/models"
)

func (c *Client) HandleUploadRequest(ctx context.Context, reqPayload []byte, sshConn *sshClientConn) error {
	c.Debugf("got request %s", string(reqPayload))

	uploadedFile, err := c.getUploadedFile(reqPayload)
	if err != nil {
		return err
	}

	destinationFileExists, err := c.filesAPI.Exist(uploadedFile.DestinationPath)
	if err != nil {
		return err
	}

	if destinationFileExists && !uploadedFile.ForceWrite {
		c.Logger.Infof("file %s already exists and not forced to be overwritten, will skip the request", uploadedFile.DestinationPath)
		return nil
	}

	fileName := path.Base(uploadedFile.DestinationPath)
	tempFilePath, err := c.copyFileToTempLocation(fileName, uploadedFile.SourceFilePath, sshConn)
	if err != nil {
		return err
	}

	err = c.prepareDestinationDir(uploadedFile.DestinationPath, uploadedFile.DestinationFileMode)
	if err != nil {
		return err
	}

	if destinationFileExists {
		c.Logger.Debugf("destination file %s already exists, will delete it", uploadedFile.DestinationPath)
		err = c.filesAPI.Remove(uploadedFile.DestinationPath)
		if err != nil {
			return err
		}
	}

	err = c.filesAPI.Rename(tempFilePath, uploadedFile.DestinationPath)
	if err != nil {
		return err
	}

	c.Logger.Debugf("moved temp file %s to the target path %s", tempFilePath, uploadedFile.DestinationPath)

	if uploadedFile.DestinationFileOwner != "" || uploadedFile.DestinationFileGroup != "" {
		err = c.filesAPI.ChangeOwner(uploadedFile.DestinationPath, uploadedFile.DestinationFileOwner, uploadedFile.DestinationFileGroup)
		if err != nil {
			return err
		}

		c.Logger.Debugf(
			"chowned file %s, %s:%s",
			uploadedFile.DestinationPath,
			uploadedFile.DestinationFileOwner,
			uploadedFile.DestinationFileGroup,
		)
	}

	return nil
}

func (c *Client) prepareDestinationDir(destinationPath string, mode os.FileMode) error {
	destinationDir := path.Dir(destinationPath)

	destinationDirWasCreated, err := c.filesAPI.CreateDirIfNotExists(destinationDir, mode)
	if err != nil {
		return err
	}
	if destinationDirWasCreated {
		c.Logger.Debugf("created destination file dir %s for the uploaded file", destinationDir)
	}

	return nil
}

func (c *Client) copyFileToTempLocation(fileName, remoteFilePath string, sshConn *sshClientConn) (tempFilePath string, err error) {
	tempDirWasCreated, err := c.filesAPI.CreateDirIfNotExists(c.configHolder.GetUploadDir(), files.DefaultMode)
	if err != nil {
		return "", err
	}
	if tempDirWasCreated {
		c.Logger.Debugf("created temp dir %s for uploaded files", c.configHolder.GetUploadDir())
	}

	tempFilePath = path.Join(c.configHolder.GetUploadDir(), fileName)

	tempFileExists, err := c.filesAPI.Exist(tempFilePath)
	if err != nil {
		return tempFilePath, err
	}

	if tempFileExists {
		c.Logger.Debugf("temp file %s already exists, will delete it", tempFilePath)
		err = c.filesAPI.Remove(tempFilePath)
		if err != nil {
			return tempFilePath, err
		}
	}

	conn := ssh.NewClient(sshConn.Connection, nil, nil)
	sftpCl, err := sftp.NewClient(conn)
	if err != nil {
		return tempFilePath, err
	}
	defer sftpCl.Close()

	remoteFile, err := sftpCl.Open(remoteFilePath)
	if err != nil {
		return tempFilePath, err
	}
	defer remoteFile.Close()

	copiedBytes, err := c.filesAPI.CreateFile(tempFilePath, remoteFile)
	if err != nil {
		return tempFilePath, err
	}
	c.Logger.Debugf("copied %d bytes from server path %s to temp path %s", copiedBytes, remoteFilePath, tempFilePath)

	return tempFilePath, nil
}

func (c *Client) getUploadedFile(reqPayload []byte) (*models.UploadedFile, error) {
	uploadedFile := new(models.UploadedFile)
	err := uploadedFile.FromBytes(reqPayload)
	if err != nil {
		return nil, err
	}
	err = uploadedFile.Validate()
	if err != nil {
		return nil, err
	}

	return uploadedFile, nil
}
