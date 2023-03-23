package chclient

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	errors2 "github.com/realvnc-labs/rport/share/errors"

	"github.com/realvnc-labs/rport/client/system"

	"github.com/pkg/sftp"

	"github.com/pkg/errors"

	"golang.org/x/crypto/ssh"

	"github.com/realvnc-labs/rport/share/files"
	"github.com/realvnc-labs/rport/share/logger"
	"github.com/realvnc-labs/rport/share/models"
)

type SourceFileProvider interface {
	Open(path string) (io.ReadCloser, error)
}

type OptionsProvider interface {
	GetUploadDir() string
	GetProtectedUploadDirs() []string
	IsFileReceptionEnabled() bool
}

type UploadManager struct {
	*logger.Logger
	FilesAPI           files.FileAPI
	OptionsProvider    OptionsProvider
	SourceFileProvider SourceFileProvider
	SysUserLookup      system.SysUserLookup
}

type SSHFileProvider struct {
	sshConn ssh.Conn
}

type SftpSession struct {
	RemoteFile io.ReadCloser
	SftpCl     *sftp.Client
}

func (ss *SftpSession) Read(p []byte) (n int, err error) {
	return ss.RemoteFile.Read(p)
}

func (ss *SftpSession) Close() error {
	errs := make([]string, 0, 2)

	err := ss.RemoteFile.Close()
	if err != nil {
		errs = append(errs, err.Error())
	}

	err = ss.SftpCl.Close()
	if err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) == 0 {
		return nil
	}

	return errors.New(strings.Join(errs, ", "))
}

func (sfp SSHFileProvider) Open(path string) (io.ReadCloser, error) {
	conn := ssh.NewClient(sfp.sshConn, nil, nil)
	sftpCl, err := sftp.NewClient(conn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to establish sftp connection to the server")
	}

	sftpFile, err := sftpCl.Open(path)

	if err != nil {
		return nil, err
	}

	return &SftpSession{
		RemoteFile: sftpFile,
		SftpCl:     sftpCl,
	}, nil
}

func NewSSHUploadManager(
	l *logger.Logger,
	filesAPI files.FileAPI,
	optionsProvider OptionsProvider,
	sshConn ssh.Conn,
	sysUserLookup system.SysUserLookup,
) *UploadManager {
	sshFileProvider := &SSHFileProvider{
		sshConn: sshConn,
	}
	return &UploadManager{
		Logger:             l,
		FilesAPI:           filesAPI,
		OptionsProvider:    optionsProvider,
		SourceFileProvider: sshFileProvider,
		SysUserLookup:      sysUserLookup,
	}
}

func (um *UploadManager) HandleUploadRequest(reqPayload []byte) (*models.UploadResponse, error) {
	if !um.OptionsProvider.IsFileReceptionEnabled() {
		err := errors2.ErrUploadsDisabled
		um.Debugf(err.Error())
		return nil, err
	}

	um.Debugf("got request %s", string(reqPayload))

	uploadedFile, err := um.getUploadedFile(reqPayload)
	if err != nil {
		return nil, err
	}

	destinationFileExists, err := um.FilesAPI.Exist(uploadedFile.DestinationPath)
	if err != nil {
		return nil, err
	}

	if !destinationFileExists || uploadedFile.ForceWrite {
		return um.handleWritingFile(uploadedFile)
	}

	fileShouldBeSynched, err := um.fileShouldBeSynched(uploadedFile)
	if err != nil {
		return nil, err
	}
	if fileShouldBeSynched {
		um.Logger.Debugf("file %s should be synched", uploadedFile.DestinationPath)
		return um.handleWritingFile(uploadedFile)
	}

	msg := fmt.Sprintf("file %s already exists, should not be synched or overwritten with force", uploadedFile.DestinationPath)
	um.Logger.Infof(msg)
	return &models.UploadResponse{
		UploadResponseShort: models.UploadResponseShort{
			ID:       uploadedFile.ID,
			Filepath: uploadedFile.DestinationPath,
		},
		Message: msg,
		Status:  "ignored",
	}, nil
}

func (um *UploadManager) fileShouldBeSynched(uploadedFile *models.UploadedFile) (bool, error) {
	if !uploadedFile.Sync {
		return false, nil
	}

	hashSumMatch, err := files.Md5HashMatch(uploadedFile.Md5Checksum, uploadedFile.DestinationPath, um.FilesAPI)

	if err != nil {
		return false, err
	}
	if !hashSumMatch {
		um.Debugf(
			"destination file %s has a different hash sum than the provided %x, the file should be synched",
			uploadedFile.DestinationPath,
			uploadedFile.Md5Checksum,
		)

		return true, nil
	}

	if uploadedFile.DestinationFileMode > 0 {
		destinationFileMode, err := um.FilesAPI.GetFileMode(uploadedFile.DestinationPath)
		if err != nil {
			return false, err
		}

		if destinationFileMode != uploadedFile.DestinationFileMode {
			um.Debugf(
				"destination file %s has a different file mode than the provided %v, the file should be synched",
				uploadedFile.DestinationPath,
				uploadedFile.DestinationFileMode,
			)

			return true, nil
		}
	}

	if uploadedFile.DestinationFileOwner != "" || uploadedFile.DestinationFileGroup != "" {
		fileOwnerMatch, err := um.fileOwnerOrGroupMatch(
			uploadedFile.DestinationPath,
			uploadedFile.DestinationFileOwner,
			uploadedFile.DestinationFileGroup,
		)
		if err != nil {
			return false, err
		}

		if !fileOwnerMatch {
			um.Debugf(
				"destination file %s has a different file owner or group than the provided ones %s:%s, ",
				uploadedFile.DestinationPath,
				uploadedFile.DestinationFileOwner,
				uploadedFile.DestinationFileGroup,
			)

			return true, nil
		}
	}

	return false, nil
}

func (um *UploadManager) handleWritingFile(uploadedFile *models.UploadedFile) (resp *models.UploadResponse, err error) {
	copiedBytes, tempFilePath, err := um.copyFileToTempLocation(
		uploadedFile.SourceFilePath,
		uploadedFile.DestinationFileMode,
		uploadedFile.Md5Checksum,
	)
	if err != nil {
		return nil, err
	}

	msgParts := []string{}

	err = um.chmodFile(tempFilePath, uploadedFile.DestinationFileMode)
	if err != nil {
		msgParts = append(msgParts, fmt.Sprintf("chmod failed: %v", err))
	}

	err = um.chownFile(tempFilePath, uploadedFile.DestinationFileOwner, uploadedFile.DestinationFileGroup)
	if err != nil {
		msgParts = append(msgParts, fmt.Sprintf("chown of %s failed: %v", tempFilePath, err))
	}

	err = um.moveFileToDestination(tempFilePath, uploadedFile)
	if err != nil {
		return nil, err
	}
	um.Logger.Debugf("copied remote file %s to the temp path %s", uploadedFile.SourceFilePath, tempFilePath)
	msgParts = append(msgParts, "file successfully copied to destination "+uploadedFile.DestinationPath)

	message := strings.Join(msgParts, ".")
	um.Logger.Debugf(message)

	return &models.UploadResponse{
		UploadResponseShort: models.UploadResponseShort{
			ID:        uploadedFile.ID,
			Filepath:  uploadedFile.DestinationPath,
			SizeBytes: copiedBytes,
		},
		Status:  "success",
		Message: message,
	}, nil
}

func (um *UploadManager) chownFile(filePath, owner, group string) (err error) {
	if owner == "" && group == "" {
		return nil
	}
	curUser, curGroup, err := um.SysUserLookup.GetCurrentUserAndGroup()
	if err != nil {
		return err
	}

	if um.ownerOrGroupMatchesCurrentUser(owner, group, curUser, curGroup) {
		um.Logger.Debugf(
			"given user %s and/or group %s match current user %s and/or group %s therefore no chown is needed",
			owner,
			group,
			curUser.Username,
			curGroup.Name,
		)
		return nil
	}

	err = um.FilesAPI.ChangeOwner(filePath, owner, group)
	if err != nil {
		return err
	}

	um.Logger.Debugf(
		"chowned file %s, %s:%s",
		filePath,
		owner,
		group,
	)

	return nil
}

func (um *UploadManager) ownerOrGroupMatchesCurrentUser(targetOwner, targetGroup string, currentUser *user.User, currentGroup *user.Group) bool {
	if targetOwner != "" && currentUser.Username == targetOwner && targetGroup == "" {
		return true
	}

	if targetOwner == "" && targetGroup != "" && currentGroup.Name == targetGroup {
		return true
	}

	return currentUser.Username == targetOwner && currentGroup.Name == targetGroup
}

func (um *UploadManager) chmodFile(path string, mode os.FileMode) (err error) {
	if mode == 0 {
		return nil
	}

	err = um.FilesAPI.ChangeMode(path, mode)
	if err != nil {
		return err
	}

	um.Logger.Debugf(
		"chmoded file %s: %v",
		path,
		mode,
	)

	return nil
}

func (um *UploadManager) moveFileToDestination(tempFilePath string, uploadedFile *models.UploadedFile) (err error) {
	err = um.prepareDestinationDir(uploadedFile.DestinationPath, uploadedFile.DestinationFileMode)
	if err != nil {
		return err
	}

	destinationFileExists, err := um.FilesAPI.Exist(uploadedFile.DestinationPath)
	if err != nil {
		return err
	}
	if destinationFileExists {
		um.Logger.Debugf("destination file %s already exists, will delete it", uploadedFile.DestinationPath)
		err = um.FilesAPI.Remove(uploadedFile.DestinationPath)
		if err != nil {
			return err
		}
	}

	err = um.FilesAPI.Rename(tempFilePath, uploadedFile.DestinationPath)
	if err != nil {
		return err
	}

	return nil
}

func (um *UploadManager) prepareDestinationDir(destinationPath string, mode os.FileMode) error {
	destinationDir := filepath.Dir(destinationPath)

	if mode == 0 {
		mode = files.DefaultMode
	}

	destinationDirWasCreated, err := um.FilesAPI.CreateDirIfNotExists(destinationDir, mode)
	if err != nil {
		return err
	}
	if destinationDirWasCreated {
		um.Logger.Debugf("created destination file dir %s for the uploaded file", destinationDir)
	}

	return nil
}

func (um *UploadManager) copyFileToTempLocation(remoteFilePath string, targetFileMode os.FileMode, expectedMd5Checksum []byte) (
	bytesCopied int64,
	tempFilePath string,
	err error,
) {
	tempFileName := filepath.Base(remoteFilePath)

	if targetFileMode == 0 {
		targetFileMode = files.DefaultMode
	}

	tempDirWasCreated, err := um.FilesAPI.CreateDirIfNotExists(um.OptionsProvider.GetUploadDir(), targetFileMode)
	if err != nil {
		return 0, "", err
	}
	if tempDirWasCreated {
		um.Logger.Debugf("created temp dir %s for uploaded files", um.OptionsProvider.GetUploadDir())
	}

	tempFilePath = filepath.Join(um.OptionsProvider.GetUploadDir(), tempFileName)

	tempFileExists, err := um.FilesAPI.Exist(tempFilePath)
	if err != nil {
		return 0, tempFilePath, err
	}

	if tempFileExists {
		um.Logger.Debugf("temp file %s already exists, will delete it", tempFilePath)
		err = um.FilesAPI.Remove(tempFilePath)
		if err != nil {
			return 0, tempFilePath, err
		}
	}

	remoteFile, err := um.SourceFileProvider.Open(remoteFilePath)
	if err != nil {
		return 0, tempFilePath, err
	}
	defer remoteFile.Close()

	copiedBytes, err := um.FilesAPI.CreateFile(tempFilePath, remoteFile)
	if err != nil {
		return 0, tempFilePath, err
	}
	um.Logger.Debugf("copied %d bytes from server path %s to temp path %s", copiedBytes, remoteFilePath, tempFilePath)

	hashSumMatch, err := files.Md5HashMatch(expectedMd5Checksum, tempFilePath, um.FilesAPI)
	if err != nil {
		return 0, tempFilePath, err
	}

	if !hashSumMatch {
		err := um.FilesAPI.Remove(tempFilePath)
		if err != nil {
			um.Logger.Errorf("failed to remove %s: %v", tempFilePath, err)
		}

		return 0,
			tempFilePath,
			fmt.Errorf(
				"md5 check failed: checksum from server %x doesn't equal the calculated checksum",
				expectedMd5Checksum,
			)
	}

	return copiedBytes, tempFilePath, nil
}

func (um *UploadManager) getUploadedFile(reqPayload []byte) (*models.UploadedFile, error) {
	uploadedFile := new(models.UploadedFile)
	err := uploadedFile.FromBytes(reqPayload)
	if err != nil {
		return nil, err
	}
	err = uploadedFile.Validate()
	if err != nil {
		return nil, err
	}

	err = uploadedFile.ValidateDestinationPath(um.OptionsProvider.GetProtectedUploadDirs(), um.Logger)
	if err != nil {
		return nil, err
	}

	return uploadedFile, nil
}

func (um *UploadManager) fileOwnerOrGroupMatch(file, owner, group string) (bool, error) {
	fileUID, fileGid, err := um.FilesAPI.GetFileOwnerAndGroup(file)
	if err != nil {
		return false, errors.Wrapf(err, "failed to read uid and gid of file %s", file)
	}

	if owner != "" {
		ownerUID, err := um.SysUserLookup.GetUIDByName(owner)
		if err != nil {
			return false, err
		}

		if fileUID != ownerUID {
			return false, nil
		}
	}

	if group != "" {
		ownerGid, err := um.SysUserLookup.GetGidByName(group)
		if err != nil {
			return false, err
		}

		if fileGid != ownerGid {
			return false, nil
		}
	}

	return true, nil
}
