package encoding

import (
	"fmt"
	"strings"

	"github.com/gogs/chardet"
	"golang.org/x/text/encoding/ianaindex"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

const (
	encoderConfidenceMinThreshold = 50
)

func ToUTF8(log *chshare.Logger, b []byte) (string, error) {
	if len(b) == 0 {
		return "", nil
	}

	d := chardet.NewTextDetector()
	r, err := d.DetectBest(b)
	if err != nil {
		return "", fmt.Errorf("failed to detect encoding: %w", err)
	}

	if r.Confidence < encoderConfidenceMinThreshold {
		log.Debugf("could not convert to UTF-8: encoding confidence is too low: threshold - %d, got - %d, charset: %s", encoderConfidenceMinThreshold, r.Confidence, r.Charset)
		return string(b), nil
	}

	if strings.ToLower(r.Charset) == "utf-8" {
		return string(b), nil
	}

	enc, err := ianaindex.IANA.Encoding(r.Charset)
	if err != nil {
		return "", fmt.Errorf("could not get encoding by IANA name %s: %w", r.Charset, err)
	}

	dec := enc.NewDecoder()
	ret, err := dec.Bytes(b)
	if err != nil {
		return "", fmt.Errorf("failed to decode from %s to UTF-8: %w", r.Charset, err)
	}

	return string(ret), nil
}
