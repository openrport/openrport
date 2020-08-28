package csr

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	chserver "github.com/cloudradar-monitoring/rport/server"
)

// GetInitStateFromFile returns an initial Client Session Repository state populated with sessions from the file.
func GetInitStateFromFile(fileName string, expiration *time.Duration) ([]*chserver.ClientSession, error) {
	log.Println("Start to get init Client Session Repository state from file.")

	file, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSR file: %v", err)
	}
	log.Printf("CSR file [%q] opened. Reading...\n", fileName)
	defer file.Close()

	return getInitState(file, expiration)
}

func getInitState(r io.Reader, expiration *time.Duration) ([]*chserver.ClientSession, error) {
	decoder := json.NewDecoder(r)
	// read array open bracket
	if _, err := decoder.Token(); err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to parse CSR data: %v", err)
	}

	var sessions []*chserver.ClientSession
	var obsolete int
	for decoder.More() {
		var session chserver.ClientSession
		if err := decoder.Decode(&session); err != nil {
			return sessions, fmt.Errorf("failed to parse client session: %v", err)
		}

		if session.Disconnected == nil || !session.Obsolete(expiration) {
			sessions = append(sessions, &session)
		} else {
			obsolete++
		}
	}

	log.Printf("Got %d and skipped %d obsolete client session(s).\n", len(sessions), obsolete)

	// mark previously connected client sessions as disconnected with current time
	now := time.Now().UTC()
	for _, session := range sessions {
		if session.Disconnected == nil {
			session.Disconnected = &now
		}
	}

	return sessions, nil
}
