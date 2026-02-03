package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
)

const nvramPrefix = "hkt_"

// nvramStore implements the hap.Store interface using router NVRAM.
// Each key is stored as a separate NVRAM variable.
//
// Commit Strategy:
// To minimise flash writes (flash has limited write cycles), we only call
// "nvram commit" when pairing data changes. Other data (uuid, keypair, schema,
// version, configHash) is written to NVRAM RAM but not committed to flash.
// This means:
//   - Normal startup: 0 flash writes
//   - Per pairing added: 1 flash write
//   - Per pairing removed: 1 flash write
//
// If power is lost before first pairing, non-pairing data is regenerated on
// next startup (new uuid/keypair). Once paired, the commit includes all pending
// changes, so keypair and pairing stay in sync.
type nvramStore struct {
	mu sync.RWMutex
}

// NewNvramStore creates a new NVRAM-backed store.
func NewNvramStore() *nvramStore {
	return &nvramStore{}
}

// nvram command wrappers - can be replaced in tests
var (
	nvramGet = func(key string) (string, error) {
		out, err := exec.Command("nvram", "get", key).Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	}

	nvramSet = func(key, value string) error {
		return exec.Command("nvram", "set", key+"="+value).Run()
	}

	nvramUnset = func(key string) error {
		return exec.Command("nvram", "unset", key).Run()
	}

	nvramCommit = func() error {
		log.Println("Committing NVRAM to flash")
		return exec.Command("nvram", "commit").Run()
	}

	nvramShow = func() (string, error) {
		out, err := exec.Command("nvram", "show").Output()
		if err != nil {
			return "", err
		}
		return string(out), nil
	}
)

// nvramKey converts a Store key to an NVRAM variable name.
// Pairing keys are converted from hex-encoded UUIDs to readable UUID strings.
func nvramKey(key string) string {
	if strings.HasSuffix(key, ".pairing") {
		hexName := strings.TrimSuffix(key, ".pairing")
		name, err := hex.DecodeString(hexName)
		if err != nil {
			return nvramPrefix + key
		}
		return nvramPrefix + "p_" + string(name)
	}
	return nvramPrefix + key
}

// Set stores a key-value pair in NVRAM.
// Text values are stored as-is, configHash is hex-encoded.
// Commits to flash only on pairing changes.
func (s *nvramStore) Set(key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	nkey := nvramKey(key)

	var encoded string
	if key == "configHash" {
		// configHash is binary (MD5 hash), hex encode it
		encoded = hex.EncodeToString(value)
	} else {
		// All other values are text (JSON, strings)
		encoded = string(value)
	}

	if err := nvramSet(nkey, encoded); err != nil {
		return fmt.Errorf("nvram set: %w", err)
	}

	// Only commit to flash when pairing data changes to reduce flash writes
	if strings.HasSuffix(key, ".pairing") {
		return nvramCommit()
	}
	return nil
}

// Get retrieves a value from NVRAM.
func (s *nvramStore) Get(key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nkey := nvramKey(key)

	value, err := nvramGet(nkey)
	if err != nil {
		return nil, err
	}
	if value == "" {
		return nil, fmt.Errorf("no entry for key %s", key)
	}

	if key == "configHash" {
		// configHash is hex-encoded binary
		return hex.DecodeString(value)
	}
	// All other values are plain text
	return []byte(value), nil
}

// Delete removes a key from NVRAM.
func (s *nvramStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	nkey := nvramKey(key)

	if err := nvramUnset(nkey); err != nil {
		return err
	}

	// Commit to flash when pairing data is deleted
	if strings.HasSuffix(key, ".pairing") {
		return nvramCommit()
	}
	return nil
}

// KeysWithSuffix returns all keys ending with the given suffix.
func (s *nvramStore) KeysWithSuffix(suffix string) (keys []string, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out, err := nvramShow()
	if err != nil {
		return nil, err
	}

	for _, line := range strings.Split(out, "\n") {
		// Skip empty lines
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		nkey := parts[0]

		if !strings.HasPrefix(nkey, nvramPrefix) {
			continue
		}

		if suffix == ".pairing" && strings.HasPrefix(nkey, nvramPrefix+"p_") {
			// Extract UUID string and reconstruct original key
			uuidStr := strings.TrimPrefix(nkey, nvramPrefix+"p_")
			originalKey := hex.EncodeToString([]byte(uuidStr)) + ".pairing"
			keys = append(keys, originalKey)
		} else {
			key := strings.TrimPrefix(nkey, nvramPrefix)
			if strings.HasSuffix(key, suffix) {
				keys = append(keys, key)
			}
		}
	}
	return
}
