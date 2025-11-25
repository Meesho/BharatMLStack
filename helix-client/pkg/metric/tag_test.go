package metric

import (
	"testing"
)

func TestNormalizeTagValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no special characters",
			input:    "simple_value",
			expected: "simple_value",
		},
		{
			name:     "colon replacement",
			input:    "key:value",
			expected: "key_value",
		},
		{
			name:     "multiple colons",
			input:    "sasl_ssl://host:port/path",
			expected: "sasl_ssl_//host_port/path",
		},
		{
			name:     "spaces replacement",
			input:    "value with spaces",
			expected: "value_with_spaces",
		},
		{
			name:     "backslashes replacement",
			input:    "path\\to\\resource",
			expected: "path_to_resource",
		},
		{
			name:     "commas replacement",
			input:    "value,with,commas",
			expected: "value_with_commas",
		},
		{
			name:     "pipes replacement",
			input:    "value|with|pipes",
			expected: "value_with_pipes",
		},
		{
			name:     "at symbols replacement",
			input:    "user@domain.com",
			expected: "user_domain.com",
		},
		{
			name:     "hash symbols replacement",
			input:    "value#with#hashes",
			expected: "value_with_hashes",
		},
		{
			name:     "kafka broker URL example",
			input:    "sasl_ssl://lkc-6wqxz3-0029.asia-southeast1-a.dom4gl1d2pl.asia-southeast1.gcp.confluent.cloud:9092/12",
			expected: "sasl_ssl_//lkc-6wqxz3-0029.asia-southeast1-a.dom4gl1d2pl.asia-southeast1.gcp.confluent.cloud_9092/12",
		},
		{
			name:     "all special characters combined",
			input:    "test:value with/spaces\\and,commas|pipes@at#hash",
			expected: "test_value_with/spaces_and_commas_pipes_at_hash",
		},
		{
			name:     "consecutive special characters",
			input:    "value::with//multiple\\\\chars",
			expected: "value__with//multiple__chars",
		},
		{
			name:     "leading and trailing special characters",
			input:    ":value:",
			expected: "_value_",
		},
		{
			name:     "only special characters",
			input:    ":@#",
			expected: "___",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeTagValue(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeTagValue(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTagAsString(t *testing.T) {
	tests := []struct {
		name     string
		tagName  string
		tagValue string
		expected string
	}{
		{
			name:     "simple tag",
			tagName:  "service",
			tagValue: "api",
			expected: "service:api",
		},
		{
			name:     "tag with special characters in value",
			tagName:  "broker",
			tagValue: "sasl_ssl://host:9092",
			expected: "broker:sasl_ssl_//host_9092",
		},
		{
			name:     "tag with spaces in value",
			tagName:  "thread",
			tagValue: "consumer thread 1",
			expected: "thread:consumer_thread_1",
		},
		{
			name:     "empty value",
			tagName:  "empty",
			tagValue: "",
			expected: "empty:",
		},
		{
			name:     "empty name",
			tagName:  "",
			tagValue: "value",
			expected: ":value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TagAsString(tt.tagName, tt.tagValue)
			if result != tt.expected {
				t.Errorf("TagAsString(%q, %q) = %q, want %q", tt.tagName, tt.tagValue, result, tt.expected)
			}
		})
	}
}
