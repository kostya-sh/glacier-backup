package main

import (
	"strings"
	"testing"
)

func TestParseConfig(t *testing.T) {
	const content = `
# vault (should be in DIR/.glacier-backup/config)
vault = test

# AWS credentials
aws_access_key_id = 1234567890ABCDEFHGHJ
aws_secret_access_key = 12345678901234567890QWERTYUIOPPASDFGHJJK

# proxy server
proxy = proxy.net
proxy_port = 8080

# max number of entries in *.db files
dbfile_size = 20

   # comment = 1`

	expected := []struct {
		key   string
		value string
	}{
		{"vault", "test"},
		{"aws_access_key_id", "1234567890ABCDEFHGHJ"},
		{"aws_secret_access_key", "12345678901234567890QWERTYUIOPPASDFGHJJK"},
		{"proxy", "proxy.net"},
		{"proxy_port", "8080"},
		{"dbfile_size", "20"},
	}

	parsed, err := parseConfig(strings.NewReader(content))
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	if len(expected) != len(parsed) {
		t.Errorf("%d entries parsed, but expected %d", len(parsed), len(expected))
	}
	for _, e := range expected {
		value := parsed[e.key]
		if value != e.value {
			t.Errorf("Key '%s': expected '%s' but was '%s'", e.key, e.value, value)
		}
	}
}

func TestParseInvalidConfig(t *testing.T) {
	invalidContents := []string{
		"invalid content",
		"key_withouth_value = ",
		" = ",
		" = empty key",
	}
	for _, content := range invalidContents {
		_, err := parseConfig(strings.NewReader(content))
		if err == nil {
			t.Errorf("Expected error for '%s'", content)
		}
	}
}
