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
	MultipleTags
	InconsistentDeviceID
	InvalidBootTime
	MissingBootTime
	OldBootTime
	NewerBootTimeFound
	InvalidBootDuration
	FastBoot
	InvalidBirthdate
	MisalignedBirthdate
	InvalidDestination
	NonEvent
	InvalidEventType
	EventTypeMismatch
	DuplicateEvent
	InconsistentMetadata
	RepeatedTransactionUUID
	MissingOnlineEvent
	MissingOfflineEvent
)

const (
	UnknownStr                 = "unknown"
	PassStr                    = "pass"
	MultipleTagsStr            = "multiple_tags"
	InconsistentDeviceIDStr    = "inconsistent_device_id"
	InvalidBootTimeStr         = "invalid_boot_time"
	MissingBootTimeStr         = "missing_boot_time"
	OldBootTimeStr             = "suspiciously_old_boot_time"
	NewerBootTimeFoundStr      = "newer_boot_time_found"
	InvalidBootDurationStr     = "invalid_boot_duration"
	FastBootStr                = "suspiciously_fast_boot"
	InvalidBirthdateStr        = "invalid_birthdate"
	MisalignedBirthdateStr     = "misaligned_birthdate"
	InvalidDestinationStr      = "invalid_destination"
	NonEventStr                = "not_an_event"
	InvalidEventTypeStr        = "invalid_event_type"
	EventTypeMismatchStr       = "event_type_mismatch"
	DuplicateEventStr          = "duplicate_event"
	InconsistentMetadataStr    = "inconsistent_metadata"
	RepeatedTransactionUUIDStr = "repeated_transaction_uuid"
	MissingOnlineEventStr      = "missing_online_event"
	MissingOfflineEventStr     = "missing_offline_event"
)

var (
	tagToString = map[Tag]string{
		Unknown:                 UnknownStr,
		Pass:                    PassStr,
		MultipleTags:            MultipleTagsStr,
		InconsistentDeviceID:    InconsistentDeviceIDStr,
		InvalidBootTime:         InvalidBootTimeStr,
		MissingBootTime:         MissingBootTimeStr,
		OldBootTime:             OldBootTimeStr,
		NewerBootTimeFound:      NewerBootTimeFoundStr,
		InvalidBootDuration:     InvalidBootDurationStr,
		FastBoot:                FastBootStr,
		InvalidBirthdate:        InvalidBirthdateStr,
		MisalignedBirthdate:     MisalignedBirthdateStr,
		InvalidDestination:      InvalidDestinationStr,
		NonEvent:                NonEventStr,
		InvalidEventType:        InvalidEventTypeStr,
		EventTypeMismatch:       EventTypeMismatchStr,
		DuplicateEvent:          DuplicateEventStr,
		InconsistentMetadata:    InconsistentMetadataStr,
		RepeatedTransactionUUID: RepeatedTransactionUUIDStr,
		MissingOnlineEvent:      MissingOnlineEventStr,
		MissingOfflineEvent:     MissingOfflineEventStr,
	}

	stringToTag = map[string]Tag{
		UnknownStr:                 Unknown,
		PassStr:                    Pass,
		MultipleTagsStr:            MultipleTags,
		InconsistentDeviceIDStr:    InconsistentDeviceID,
		InvalidBootTimeStr:         InvalidBootTime,
		MissingBootTimeStr:         MissingBootTime,
		OldBootTimeStr:             OldBootTime,
		NewerBootTimeFoundStr:      NewerBootTimeFound,
		InvalidBootDurationStr:     InvalidBootDuration,
		FastBootStr:                FastBoot,
		InvalidBirthdateStr:        InvalidBirthdate,
		MisalignedBirthdateStr:     MisalignedBirthdate,
		InvalidDestinationStr:      InvalidDestination,
		NonEventStr:                NonEvent,
		InvalidEventTypeStr:        InvalidEventType,
		EventTypeMismatchStr:       EventTypeMismatch,
		DuplicateEventStr:          DuplicateEvent,
		InconsistentMetadataStr:    InconsistentMetadata,
		RepeatedTransactionUUIDStr: RepeatedTransactionUUID,
		MissingOnlineEventStr:      MissingOnlineEvent,
		MissingOfflineEventStr:     MissingOfflineEvent,
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
