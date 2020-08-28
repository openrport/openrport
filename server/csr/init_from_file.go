package csr

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	chserver "github.com/cloudradar-monitoring/rport/server"
)

// InitAndPopulateFromFile returns a Client Session Repository populated with sessions from the file.
func InitAndPopulateFromFile(fileName string, expiration *time.Duration) (chserver.ClientSessionRepository, error) {
	log.Println("Start to init Client Session Repository from file.")
	sessions, err := readNonObsoleteFromFile(fileName, expiration)
	// proceed further in case some sessions were parsed successfully and return the error at the end

	// mark previously connected client sessions as disconnected with current time
	now := time.Now()
	for _, session := range sessions {
		if session.Disconnected == nil {
			session.Disconnected = &now
		}
	}

	return *chserver.NewSessionRepository(sessions, expiration), err
}

func readNonObsoleteFromFile(fileName string, expiration *time.Duration) ([]*chserver.ClientSession, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSR file: %v", err)
	}
	log.Printf("CSR file [%q] opened. Parsing...\n", fileName)
	defer file.Close()

	decoder := json.NewDecoder(file)
	// read array open bracket
	if _, err := decoder.Token(); err != nil {
		return nil, fmt.Errorf("failed to parse CSR file: %v", err)
	}

	var sessions []*chserver.ClientSession
	var obsolete int
	for decoder.More() {
		var session chserver.ClientSession
		if err := decoder.Decode(&session); err != nil {
			return sessions, fmt.Errorf("failed to parse CSR data: %v", err)
		}

		if session.Disconnected == nil || !session.Obsolete(expiration) {
			sessions = append(sessions, &session)
		} else {
			obsolete++
		}
	}

	log.Printf("Got %d and skipped %d obsolete client session(s) from CSR file.\n", len(sessions), obsolete)

	return sessions, nil
}
