package errors

import "errors"

var ErrUploadsDisabled = errors.New("uploads are disabled on this client, check [file-reception] enabled option")
