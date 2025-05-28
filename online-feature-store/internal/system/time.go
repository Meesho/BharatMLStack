package system

import (
	"fmt"
	"time"
)

// Add at the top of the file with other package-level declarations
var timeNow = time.Now

const (
	yearBits = 14  // Current year (when data is persisted)
	dayBits  = 9   // Days (0-512)
	secsBits = 17  // Seconds in day
	maxDays  = 513 // Maximum days allowed

	dayMask  = uint64((1 << dayBits) - 1)
	secsMask = uint64((1 << secsBits) - 1)
)

// EncodeExpiry encodes expiry timestamp into 5 bytes following format:
// [14 bits persistence year][9 bits days][17 bits seconds]
func EncodeExpiry(expiryEpoch uint64) ([]byte, error) {
	if expiryEpoch == 0 {
		// set expiry to 0 (lifetime ttl)
		result := make([]byte, 5)
		return result, nil
	}
	now := timeNow().UTC()
	currentYear := now.Year()

	// Get start of current year in epoch seconds
	startOfYear := time.Date(currentYear, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

	// Validate expiry time
	if expiryEpoch <= uint64(now.Unix()) {
		return nil, fmt.Errorf("expiry time must be in future")
	}

	maxAllowed := uint64(now.Unix()) + uint64(maxDays*24*60*60)
	if expiryEpoch > maxAllowed {
		return nil, fmt.Errorf("expiry time cannot be more than 513 days from now")
	}

	// Calculate rest epoch
	restEpoch := int64(expiryEpoch) - startOfYear

	// Calculate complete days and remaining seconds
	completeDays := uint64(0)
	remainingSecs := uint64(0)

	if restEpoch > 0 {
		completeDays = uint64(restEpoch) / (24 * 60 * 60)
		remainingSecs = uint64(restEpoch) % (24 * 60 * 60)
	} else {
		remainingSecs = uint64(-restEpoch)
	}

	// Pack the values
	yearPart := uint64(currentYear) << (dayBits + secsBits)
	dayPart := (completeDays & dayMask) << secsBits
	secsPart := remainingSecs & secsMask

	packed := yearPart | dayPart | secsPart

	// Convert to 5 bytes using system byte order
	result := make([]byte, 5)
	ByteOrder.PutUint32(result[0:4], uint32(packed>>8))
	result[4] = byte(packed)

	return result, nil
}

// IsExpired checks if the given 5-byte expiry timestamp has expired
func IsExpired(expiryBytes []byte) bool {
	if len(expiryBytes) != 5 {
		return true // Invalid format, consider expired
	}
	// Check if the expiry is set to 0 (lifetime ttl)
	fullBytes := append(make([]byte, 3), expiryBytes...) // Prepend 3 bytes of 0 to make it 8 bytes
	if ByteOrder.Uint64(fullBytes) == 0 {
		return false // Lifetime ttl, not expired
	}

	// Reconstruct 40-bit packed value
	packed := uint64(ByteOrder.Uint32(expiryBytes[0:4]))<<8 | uint64(expiryBytes[4])

	// Extract parts
	storedYear := int(packed >> (dayBits + secsBits))
	storedDays := int((packed >> secsBits) & dayMask)
	storedSecs := int(packed & secsMask)

	// Get current time components
	now := timeNow().UTC()

	// Calculate expiry time
	startOfStoredYear := time.Date(storedYear, 1, 1, 0, 0, 0, 0, time.UTC)
	expiryTime := startOfStoredYear.AddDate(0, 0, storedDays)
	expiryTime = expiryTime.Add(time.Duration(storedSecs) * time.Second)

	return now.After(expiryTime)
}

func DecodeExpiry(expiryBytes []byte) (uint64, error) {
	if len(expiryBytes) != 5 {
		return 0, fmt.Errorf("invalid expiry bytes length")
	}

	packed := uint64(ByteOrder.Uint32(expiryBytes[0:4]))<<8 | uint64(expiryBytes[4])

	year := int(packed >> (dayBits + secsBits))
	days := int((packed >> secsBits) & dayMask)
	secs := int(packed & secsMask)
	//Epoch of the year
	epochOfYear := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	//Epoch of the days
	epochOfDays := epochOfYear + int64(days*24*60*60)
	//Epoch of the seconds
	epochOfSecs := epochOfDays + int64(secs)
	return uint64(epochOfSecs), nil
}
