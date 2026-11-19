package checks

import (
	"encoding/json"
	"testing"
)

func TestCommandCheckValidation(t *testing.T) {
	// 1. Prohibited command with shell injection
	badConfigs := []string{
		`{"command": "rm -rf /"}`,
		`{"command": "bash -c 'whoami'"}`,
		`{"command": "/usr/bin/uptime; curl evil.com"}`,
		`{"command": "/usr/bin/uptime | sh"}`,
		`{"command": "../../../bin/sh"}`,
		`{"command": "/etc/sentrix/scripts/../../bin/sh"}`,
	}

	for _, bad := range badConfigs {
		err := validateCheckConfig("COMMAND", json.RawMessage(bad))
		if err == nil {
			t.Errorf("Expected validation failure for dangerous command config %s, got nil", bad)
		}
	}

	// 2. Allowed valid commands
	goodConfigs := []string{
		`{"command": "/usr/bin/uptime"}`,
		`{"command": "/bin/df"}`,
		`{"command": "/usr/bin/free"}`,
		`{"command": "/usr/bin/vmstat"}`,
		`{"command": "/etc/sentrix/scripts/disk_check.sh"}`,
		`{"command": "/usr/local/bin/sentrix-probe-nginx"}`,
	}

	for _, good := range goodConfigs {
		err := validateCheckConfig("COMMAND", json.RawMessage(good))
		if err != nil {
			t.Errorf("Expected valid command config %s to succeed, got error: %v", good, err)
		}
	}
}
