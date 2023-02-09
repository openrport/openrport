package simple_client_connects_test

import (
	"encoding/json"
	"testing"

	"github.com/KonradKuznicki/must"
	"github.com/stretchr/testify/assert"

	"github.com/cloudradar-monitoring/rport/bdd/helpers"
)

type TagsAndLabels struct {
	Tags   []string `json:"tags"`
	Labels []string `json:"labels"`
}

type Rsp struct {
	Data []TagsAndLabels `json:"data"`
}

func TestClientHasTagsAndLabelsSearchable(t *testing.T) {

	rd, rc := helpers.StartClientAndServerAndWaitForConnection(t)
	defer func() {
		rd.Process.Kill()
		rc.Process.Kill()
	}()

	requestURL := "http://localhost:3000/api/v1/clients?fields[clients]=tags,labels"
	data := helpers.CallAPIGET(t, requestURL)

	var structured Rsp

	must.Must0(json.Unmarshal([]byte(data), &structured))

	assert.Equal(t, structured, Rsp{Data: []TagsAndLabels{{Tags: []string{"task-vm", "vm"}}}})

}
