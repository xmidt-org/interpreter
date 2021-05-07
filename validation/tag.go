package validation

import "strings"

// Tag is an enum used to flag the problems with an event.
type Tag int

func (t Tag) String() string {
	if val, ok := tagToString[t]; ok {
		return val
	}

	return "unknown"
}

const (
	Unknown Tag = iota
	Pass
	InconsistentDeviceID
	InvalidBootTime
	MissingBootTime
	OldBootTime
	OutdatedBootTime
	InvalidBootDuration
	FastBoot
	InvalidBirthdate
	MisalignedBirthdate
	InvalidDestination
	NonEvent
	InvalidEventType
	EventTypeMismatch
	DuplicateEvent
)

var (
	tagToString = map[Tag]string{
		Unknown:              "unknown",
		Pass:                 "pass",
		InconsistentDeviceID: "inconsistent_device_id",
		InvalidBootTime:      "invalid_boot_time",
		MissingBootTime:      "missing_boot_time",
		OldBootTime:          "suspiciously_old_boot_time",
		OutdatedBootTime:     "outdated_boot_time",
		InvalidBootDuration:  "invalid_boot_duration",
		FastBoot:             "suspiciously_fast_boot",
		InvalidBirthdate:     "invalid_birthdate",
		MisalignedBirthdate:  "misaligned_birthdate",
		InvalidDestination:   "invalid_destination",
		NonEvent:             "not_an_event",
		InvalidEventType:     "invalid_event_type",
		EventTypeMismatch:    "event_type_mismatch",
		DuplicateEvent:       "duplicate_event",
	}

	stringToTag = map[string]Tag{
		"unknown":                    Unknown,
		"pass":                       Pass,
		"inconsistent_device_id":     InconsistentDeviceID,
		"invalid_boot_time":          InvalidBootTime,
		"missing_boot_time":          MissingBootTime,
		"suspiciously_old_boot_time": OldBootTime,
		"outdated_boot_time":         OutdatedBootTime,
		"invalid_boot_duration":      InvalidBootDuration,
		"suspiciously_fast_boot":     FastBoot,
		"invalid_birthdate":          InvalidBirthdate,
		"misaligned_birthdate":       MisalignedBirthdate,
		"invalid_destination":        InvalidDestination,
		"not_an_event":               NonEvent,
		"invalid_event_type":         InvalidEventType,
		"event_type_mismatch":        EventTypeMismatch,
		"duplicate_event":            DuplicateEvent,
	}
)

// ParseTag is used to convert a string to a Tag. Returns Unknown if the string is not known.
func ParseTag(str string) Tag {
	str = strings.Replace(strings.ToLower(str), " ", "_", -1)
	if val, ok := stringToTag[str]; ok {
		return val
	}

	return Unknown
}
