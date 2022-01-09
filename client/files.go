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

	uploadedFile := new(models.UploadedFile)
	err := uploadedFile.FromBytes(reqPayload)
	if err != nil {
		return err
	}
	err = uploadedFile.Validate()
	if err != nil {
		return err
	}

	destinationFileExists, err := files.FileOrDirExists(uploadedFile.DestinationPath)
	if err != nil {
		return err
	}

	if destinationFileExists && !uploadedFile.ForceWrite {
		c.Logger.Infof("file %s already exists and not forced to be overwritten, will skip the request", uploadedFile.DestinationPath)
		return nil
	}

	wasCreated, err := files.CreateDirIfNotExists(c.configHolder.GetUploadDir(), files.DefaultMode)
	if err != nil {
		return err
	}
	if wasCreated {
		c.Logger.Debugf("created temp dir %s for uploaded files", c.configHolder.GetUploadDir())
	}

	fileName := path.Base(uploadedFile.DestinationPath)
	tempFileNamePath := path.Join(c.configHolder.GetUploadDir(), fileName)

	tempFileExists, err := files.FileOrDirExists(tempFileNamePath)
	if err != nil {
		return err
	}

	if tempFileExists {
		c.Logger.Debugf("temp file %s already exists, will delete it", tempFileNamePath)
		err = os.Remove(tempFileNamePath)
		if err != nil {
			return err
		}
	}

	conn := ssh.NewClient(sshConn.Connection, nil, nil)
	sftpCl, err := sftp.NewClient(conn)
	if err != nil {
		return err
	}
	defer sftpCl.Close()

	remoteFile, err := sftpCl.Open(uploadedFile.SourceFilePath)
	if err != nil {
		return err
	}
	defer remoteFile.Close()

	copiedBytes, err := files.CopyFileToDestination(tempFileNamePath, remoteFile, c.Logger)
	if err != nil {
		return err
	}

	c.Logger.Debugf("copied %d bytes from server path %s to temp path %s", copiedBytes, uploadedFile.SourceFilePath, tempFileNamePath)

	wasCreated, err = files.CreateDirIfNotExists(path.Dir(uploadedFile.DestinationPath), files.DefaultMode)
	if err != nil {
		return err
	}
	if wasCreated {
		c.Logger.Debugf("created destination file dir %s for the uploaded file", path.Dir(uploadedFile.DestinationPath))
	}

	if destinationFileExists {
		c.Logger.Debugf("destination file %s already exists, will delete it", uploadedFile.DestinationPath)
		err = os.Remove(uploadedFile.DestinationPath)
		if err != nil {
			return err
		}
	}

	err = os.Rename(tempFileNamePath, uploadedFile.DestinationPath)
	if err != nil {
		return err
	}

	c.Logger.Debugf("moved temp file %s to the target path %s", tempFileNamePath, uploadedFile.DestinationPath)

	err = files.ChangeOwner(uploadedFile, c.Logger)
	if err != nil {
		return err
	}

	return nil
}
