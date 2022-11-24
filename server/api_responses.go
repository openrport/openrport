package chserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/cloudradar-monitoring/rport/server/api"
	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
	"github.com/cloudradar-monitoring/rport/share/logger"
)

var (
	// this will be used by default for tests, but otherwise should be overridden during server init
	errLog = logger.NewLogger("api-error-response", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)
)

func SetAPIResponsesErrorLog(l *logger.Logger) {
	errLog = l
}

func writeErrorPayloadLog(errPayload api.ErrorPayload) {
	if errLog != nil && errLog.Level == logger.LogLevelDebug {
		errLog.Debugf("payload: %+v", errPayload)
	}
}

func (al *APIListener) writeJSONResponse(w http.ResponseWriter, statusCode int, response interface{}) {
	b, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(statusCode)
	if _, err := w.Write(b); err != nil {
		al.Errorf("error writing response: %s", err)
	}
}

func (al *APIListener) jsonErrorResponse(w http.ResponseWriter, statusCode int, err error) {
	errPayload := api.NewErrAPIPayloadFromError(err, "", "")
	writeErrorPayloadLog(errPayload)
	al.writeJSONResponse(w, statusCode, errPayload)
}

func (al *APIListener) jsonError(w http.ResponseWriter, err error) {
	statusCode := http.StatusInternalServerError
	errCode := ""
	var apiErr errors2.APIError
	var apiErrs errors2.APIErrors
	switch {
	case errors.As(err, &apiErr):
		statusCode = apiErr.HTTPStatus
		errCode = apiErr.ErrCode
	case errors.As(err, &apiErrs):
		if len(apiErrs) > 0 {
			statusCode = apiErrs[0].HTTPStatus
			errCode = apiErrs[0].ErrCode
		}
	}

	errPayload := api.NewErrAPIPayloadFromError(err, errCode, "")
	writeErrorPayloadLog(errPayload)
	al.writeJSONResponse(w, statusCode, errPayload)
}

func (al *APIListener) jsonErrorResponseWithErrCode(w http.ResponseWriter, statusCode int, errCode, title string) {
	errPayload := api.NewErrAPIPayloadFromMessage(errCode, title, "")
	writeErrorPayloadLog(errPayload)
	al.writeJSONResponse(w, statusCode, errPayload)
}

func (al *APIListener) jsonErrorResponseWithTitle(w http.ResponseWriter, statusCode int, title string) {
	errPayload := api.NewErrAPIPayloadFromMessage("", title, "")
	writeErrorPayloadLog(errPayload)
	al.writeJSONResponse(w, statusCode, errPayload)
}

func (al *APIListener) jsonErrorResponseWithDetail(w http.ResponseWriter, statusCode int, errCode, title, detail string) {
	errPayload := api.NewErrAPIPayloadFromMessage(errCode, title, detail)
	writeErrorPayloadLog(errPayload)
	al.writeJSONResponse(w, statusCode, errPayload)
}

func (al *APIListener) jsonErrorResponseWithError(w http.ResponseWriter, statusCode int, title string, err error) {
	var detail string
	if err != nil {
		detail = err.Error()
	}
	errPayload := api.NewErrAPIPayloadFromMessage("", title, detail)
	writeErrorPayloadLog(errPayload)
	al.writeJSONResponse(w, statusCode, errPayload)
}
