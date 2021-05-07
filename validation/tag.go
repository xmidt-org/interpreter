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
	MissingBootTime
	OldBootTime
	InvalidBootTime
	FastBoot
	InvalidBirthdate
	MisalignedBirthdate
	InvalidDestination
	WrongEventType
)

var (
	tagToString = map[Tag]string{
		Pass:                 "pass",
		InconsistentDeviceID: "inconsistent_device_id",
		MissingBootTime:      "missing_boot_time",
		OldBootTime:          "old_boot_time",
		InvalidBootTime:      "invalid_boot_time",
		FastBoot:             "suspiciously_fast_boot",
		InvalidBirthdate:     "invalid_birthdate",
		MisalignedBirthdate:  "misaligned_birthdate",
		Unknown:              "unknown",
		InvalidDestination:   "invalid_destination",
		WrongEventType:       "wrong_event_type",
	}

	stringToTag = map[string]Tag{
		"pass":                   Pass,
		"inconsistent_device_id": InconsistentDeviceID,
		"missing_boot_time":      MissingBootTime,
		"old_boot_time":          OldBootTime,
		"invalid_boot_time":      InvalidBootTime,
		"invalid_birthdate":      InvalidBirthdate,
		"misaligned_birthdate":   MisalignedBirthdate,
		"suspiciously_fast_boot": FastBoot,
		"unknown":                Unknown,
		"invalid_destination":    InvalidDestination,
		"wrong_event_type":       WrongEventType,
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
