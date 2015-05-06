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

func TestMergeAndValidateConfigs(t *testing.T) {
	testData := []struct {
		main     map[string]string
		fallback map[string]string
		expected config
	}{
		{
			// all form main
			map[string]string{
				"vault":                 "v",
				"aws_access_key_id":     "key",
				"aws_secret_access_key": "pwd",
				"proxy":                 "proxy",
				"proxy_port":            "80",
				"region":                "r",
				"dbfile_size":           "25"},
			map[string]string{
				"vault":                 "v2",
				"aws_access_key_id":     "key2",
				"aws_secret_access_key": "pwd2",
				"proxy":                 "proxy2",
				"proxy_port":            "81",
				"region":                "r2",
				"dbfile_size":           "26"},
			config{"v", "key", "pwd", "proxy", 80, "r", 25},
		},
		{
			// all form fallback
			map[string]string{},
			map[string]string{
				"vault":                 "v2",
				"aws_access_key_id":     "key2",
				"aws_secret_access_key": "pwd2",
				"proxy":                 "proxy2",
				"proxy_port":            "81",
				"region":                "r2",
				"dbfile_size":           "26"},
			config{"v2", "key2", "pwd2", "proxy2", 81, "r2", 26},
		},
		{
			// mix main and fallback
			map[string]string{
				"vault":       "v",
				"proxy":       "proxy",
				"proxy_port":  "80",
				"region":      "r",
				"dbfile_size": "25"},
			map[string]string{
				"vault":                 "v2",
				"aws_access_key_id":     "key2",
				"aws_secret_access_key": "pwd2",
				"proxy":                 "proxy2",
				"proxy_port":            "81",
				"region":                "r2",
				"dbfile_size":           "26"},
			config{"v", "key2", "pwd2", "proxy", 80, "r", 25},
		},
		{
			// default values
			map[string]string{},
			map[string]string{
				"vault":                 "v2",
				"aws_access_key_id":     "key2",
				"aws_secret_access_key": "pwd2"},
			config{"v2", "key2", "pwd2", "", 0, "us-east-1", 20},
		},
	}

	for _, e := range testData {
		actual, err := mergeAndValidateConfigs(e.main, e.fallback)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if *actual != e.expected {
			t.Errorf("Expected %+v but was %+v", e.expected, *actual)
		}
	}
}

func TestMergeAndValidateInvalidConfigs(t *testing.T) {
	invalidConfigs := []map[string]string{
		map[string]string{"vault": "v", "aws_access_key_id": "key", "aws_secret_access_key": "pwd", "proxy_port": "nan"},
		map[string]string{"vault": "v", "aws_access_key_id": "key", "aws_secret_access_key": "pwd", "dbfile_size": "nan"},
		map[string]string{"aws_access_key_id": "key", "aws_secret_access_key": "pwd"},
		map[string]string{"vault": "v", "aws_secret_access_key": "pwd"},
		map[string]string{"vault": "v", "aws_access_key_id": "key"},
	}

	for _, configMap := range invalidConfigs {
		_, err := mergeAndValidateConfigs(configMap, map[string]string{})
		if err == nil {
			t.Errorf("Expected error for main config %+v", configMap)
		}
		_, err = mergeAndValidateConfigs(map[string]string{}, configMap)
		if err == nil {
			t.Errorf("Expected error for fallback config %+v", configMap)
		}
	}
}
