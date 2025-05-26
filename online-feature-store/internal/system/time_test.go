package system

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEncodeExpiry(t *testing.T) {
	Init()
	// Save the original function and restore after all tests
	originalNow := timeNow
	defer func() { timeNow = originalNow }()

	tests := []struct {
		name        string
		setupTime   time.Time // Time to set as "now"
		expiryEpoch uint64    // Input expiry epoch
		wantErr     bool      // Whether we expect an error
		errMsg      string    // Expected error message (if wantErr is true)
	}{
		{
			name:        "Valid future expiry within same year",
			setupTime:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expiryEpoch: uint64(time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC).Unix()),
			wantErr:     false,
		},
		{
			name:        "Valid future expiry crossing year boundary",
			setupTime:   time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
			expiryEpoch: uint64(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Unix()),
			wantErr:     false,
		},
		{
			name:        "Past expiry",
			setupTime:   time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			expiryEpoch: uint64(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix()),
			wantErr:     true,
			errMsg:      "expiry time must be in future",
		},
		{
			name:        "Expiry too far in future (> 513 days)",
			setupTime:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expiryEpoch: uint64(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Add(514 * 24 * time.Hour).Unix()),
			wantErr:     true,
			errMsg:      "expiry time cannot be more than 513 days from now",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock current time
			timeNow = func() time.Time {
				return tt.setupTime
			}

			got, err := EncodeExpiry(tt.expiryEpoch)
			if tt.wantErr {
				if err == nil {
					t.Errorf("EncodeExpiry() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("EncodeExpiry() error = %v, want %v", err, tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("EncodeExpiry() unexpected error = %v", err)
				return
			}
			if len(got) != 5 {
				t.Errorf("EncodeExpiry() returned %d bytes, want 5", len(got))
			}
		})
	}
}

func TestIsExpired(t *testing.T) {
	Init()
	// Save the original function and restore after all tests
	originalNow := timeNow
	defer func() { timeNow = originalNow }()

	tests := []struct {
		name      string
		setupTime time.Time // Time to set as "now"
		setup     func() []byte
		want      bool
	}{
		{
			name:      "Invalid byte length",
			setupTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			setup: func() []byte {
				return []byte{1, 2, 3} // Invalid length
			},
			want: true,
		},
		{
			name:      "Expired - same year, earlier day",
			setupTime: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			setup: func() []byte {
				// Mock time to encode
				timeNow = func() time.Time {
					return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
				}

				b, _ := EncodeExpiry(uint64(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).Unix()))
				return b
			},
			want: true,
		},
		{
			name:      "Not expired - same year, future day",
			setupTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			setup: func() []byte {
				timeNow = func() time.Time {
					return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
				}

				b, _ := EncodeExpiry(uint64(time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC).Unix()))
				return b
			},
			want: false,
		},
		{
			name:      "Not expired - crossing year boundary",
			setupTime: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			setup: func() []byte {
				timeNow = func() time.Time {
					return time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
				}

				b, _ := EncodeExpiry(uint64(time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC).Unix()))
				return b
			},
			want: false,
		},
		{
			name:      "Same day - compare seconds (expired)",
			setupTime: time.Date(2024, 1, 1, 15, 0, 0, 0, time.UTC),
			setup: func() []byte {
				timeNow = func() time.Time {
					return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
				}

				b, _ := EncodeExpiry(uint64(time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC).Unix()))
				return b
			},
			want: true,
		},
		{
			name:      "Maximum TTL test (513 days)",
			setupTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			setup: func() []byte {
				timeNow = func() time.Time {
					return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
				}

				b, _ := EncodeExpiry(uint64(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Add(513 * 24 * time.Hour).Unix()))
				return b
			},
			want: false,
		},
		{
			name:      "Leap year handling",
			setupTime: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			setup: func() []byte {
				timeNow = func() time.Time {
					return time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC) // 2024 is a leap year
				}

				b, _ := EncodeExpiry(uint64(time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC).Unix()))
				return b
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := tt.setup()
			// Mock current time
			timeNow = func() time.Time {
				return tt.setupTime
			}

			if got := IsExpired(b); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

//
//func TestDecodeExpiry(t *testing.T) {
//	Init()
//	// Save the original function and restore after all tests
//	originalNow := timeNow
//	defer func() { timeNow = originalNow }()
//
//	tests := []struct {
//		name      string
//		setupTime time.Time // Time to set as "now"
//		input     []byte    // Input bytes
//		want      uint64    // Expected Unix timestamp
//		wantErr   bool
//		errMsg    string
//	}{
//		{
//			name:      "Valid expiry within same year",
//			setupTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
//			input:     []byte{0x00, 0x1F, 0x00, 0x00, 0x00}, // Feb 1, 2024
//			want:      uint64(time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC).Unix()),
//			wantErr:   false,
//		},
//		{
//			name:      "Valid expiry crossing year boundary",
//			setupTime: time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
//			input:     []byte{0x00, 0x01, 0x00, 0x00, 0x00}, // Jan 1, 2025
//			want:      uint64(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Unix()),
//			wantErr:   false,
//		},
//		{
//			name:      "Invalid byte length",
//			setupTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
//			input:     []byte{0x00, 0x1F, 0x00}, // Too short
//			wantErr:   true,
//			errMsg:    "invalid expiry bytes length",
//		},
//		{
//			name:      "Invalid day of year",
//			setupTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
//			input:     []byte{0x00, 0xFF, 0x00, 0x00, 0x00}, // Day 255 (invalid)
//			wantErr:   true,
//			errMsg:    "invalid day of year",
//		},
//		{
//			name:      "Leap year handling",
//			setupTime: time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC),
//			input:     []byte{0x00, 0x3C, 0x00, 0x00, 0x00}, // Day 60 (Feb 29, 2024)
//			want:      uint64(time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC).Unix()),
//			wantErr:   false,
//		},
//		{
//			name:      "Same day, different seconds",
//			setupTime: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
//			input:     []byte{0x00, 0x01, 0x00, 0xFF, 0xFF}, // Jan 1, late in day
//			want:      uint64(time.Date(2024, 1, 1, 23, 59, 59, 0, time.UTC).Unix()),
//			wantErr:   false,
//		},
//		{
//			name:      "Maximum seconds in day",
//			setupTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
//			input:     []byte{0x00, 0x01, 0x00, 0xFF, 0xFF}, // Max seconds
//			want:      uint64(time.Date(2024, 1, 1, 23, 59, 59, 0, time.UTC).Unix()),
//			wantErr:   false,
//		},
//		{
//			name:      "Invalid seconds value",
//			setupTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
//			input:     []byte{0x00, 0x01, 0x01, 0xFF, 0xFF}, // Invalid seconds encoding
//			wantErr:   true,
//			errMsg:    "invalid seconds value",
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// Mock current time
//			timeNow = func() time.Time {
//				return tt.setupTime
//			}
//
//			got, err := DecodeExpiry(tt.input)
//			if tt.wantErr {
//				assert.Error(t, err)
//				if tt.errMsg != "" {
//					assert.Contains(t, err.Error(), tt.errMsg)
//				}
//				return
//			}
//
//			assert.NoError(t, err)
//			assert.Equal(t, tt.want, got)
//
//			// Additional verification: encode the decoded value and compare
//			if !tt.wantErr {
//				encoded, err := EncodeExpiry(got)
//				assert.NoError(t, err)
//				decodedAgain, err := DecodeExpiry(encoded)
//				assert.NoError(t, err)
//				assert.Equal(t, got, decodedAgain, "encode-decode cycle should preserve value")
//			}
//		})
//	}
//}

func TestEncodeDecodeExpiry(t *testing.T) {
	Init()
	// Save the original function and restore after all tests
	originalNow := timeNow
	defer func() { timeNow = originalNow }()

	tests := []struct {
		name      string
		inputTime time.Time
		setupTime time.Time // Time to set as "now"
		wantErr   bool
	}{
		{
			name:      "start of year",
			setupTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			inputTime: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "end of year",
			setupTime: time.Date(2024, 12, 30, 0, 0, 0, 0, time.UTC),
			inputTime: time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
		},
		{
			name:      "leap day",
			setupTime: time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC),
			inputTime: time.Date(2024, 2, 29, 12, 30, 45, 0, time.UTC),
		},
		{
			name:      "non-leap year",
			setupTime: time.Date(2025, 2, 27, 0, 0, 0, 0, time.UTC),
			inputTime: time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "middle of year with time",
			setupTime: time.Date(2024, 7, 14, 0, 0, 0, 0, time.UTC),
			inputTime: time.Date(2024, 7, 15, 13, 45, 30, 0, time.UTC),
		},
		{
			name:      "last second of day",
			setupTime: time.Date(2024, 3, 14, 0, 0, 0, 0, time.UTC),
			inputTime: time.Date(2024, 3, 15, 23, 59, 59, 0, time.UTC),
		},
		{
			name:      "first second of day",
			setupTime: time.Date(2024, 3, 14, 0, 0, 0, 0, time.UTC),
			inputTime: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock current time
			timeNow = func() time.Time {
				return tt.setupTime
			}

			epochTime := uint64(tt.inputTime.Unix())

			encoded, err := EncodeExpiry(epochTime)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Len(t, encoded, 5, "encoded bytes should be 5 bytes")

			// Decode back
			decoded, err := DecodeExpiry(encoded)
			assert.NoError(t, err)

			decodedTime := time.Unix(int64(decoded), 0).UTC()
			assert.Equal(t, epochTime, decoded, "decoded time mismatch")
			assert.Equal(t, tt.inputTime.Unix(), decodedTime.Unix(),
				"Times don't match. Input: %v, Got: %v",
				tt.inputTime.Format(time.RFC3339),
				decodedTime.Format(time.RFC3339))
		})
	}
}
