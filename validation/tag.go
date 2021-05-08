package validation

import "strings"

// Tag is an enum used to flag the problems with an event.
type Tag int

func (t Tag) String() string {
	if val, ok := tagToString[t]; ok {
		return val
	}

	return UnknownStr
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

const (
	UnknownStr              = "unknown"
	PassStr                 = "pass"
	InconsistentDeviceIDStr = "inconsistent_device_id"
	InvalidBootTimeStr      = "invalid_boot_time"
	MissingBootTimeStr      = "missing_boot_time"
	OldBootTimeStr          = "suspiciously_old_boot_time"
	OutdatedBootTimeStr     = "outdated_boot_time"
	InvalidBootDurationStr  = "invalid_boot_duration"
	FastBootStr             = "suspiciously_fast_boot"
	InvalidBirthdateStr     = "invalid_birthdate"
	MisalignedBirthdateStr  = "misaligned_birthdate"
	InvalidDestinationStr   = "invalid_destination"
	NonEventStr             = "not_an_event"
	InvalidEventTypeStr     = "invalid_event_type"
	EventTypeMismatchStr    = "event_type_mismatch"
	DuplicateEventStr       = "duplicate_event"
)

var (
	tagToString = map[Tag]string{
		Unknown:              UnknownStr,
		Pass:                 PassStr,
		InconsistentDeviceID: InconsistentDeviceIDStr,
		InvalidBootTime:      InvalidBootTimeStr,
		MissingBootTime:      MissingBootTimeStr,
		OldBootTime:          OldBootTimeStr,
		OutdatedBootTime:     OutdatedBootTimeStr,
		InvalidBootDuration:  InvalidBootDurationStr,
		FastBoot:             FastBootStr,
		InvalidBirthdate:     InvalidBirthdateStr,
		MisalignedBirthdate:  MisalignedBirthdateStr,
		InvalidDestination:   InvalidDestinationStr,
		NonEvent:             NonEventStr,
		InvalidEventType:     InvalidEventTypeStr,
		EventTypeMismatch:    EventTypeMismatchStr,
		DuplicateEvent:       DuplicateEventStr,
	}

	stringToTag = map[string]Tag{
		UnknownStr:              Unknown,
		PassStr:                 Pass,
		InconsistentDeviceIDStr: InconsistentDeviceID,
		InvalidBootTimeStr:      InvalidBootTime,
		MissingBootTimeStr:      MissingBootTime,
		OldBootTimeStr:          OldBootTime,
		OutdatedBootTimeStr:     OutdatedBootTime,
		InvalidBootDurationStr:  InvalidBootDuration,
		FastBootStr:             FastBoot,
		InvalidBirthdateStr:     InvalidBirthdate,
		MisalignedBirthdateStr:  MisalignedBirthdate,
		InvalidDestinationStr:   InvalidDestination,
		NonEventStr:             NonEvent,
		InvalidEventTypeStr:     InvalidEventType,
		EventTypeMismatchStr:    EventTypeMismatch,
		DuplicateEventStr:       DuplicateEvent,
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
