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
	MultipleTags            // used for multiple errors or cases where there are multiple error tags
	MissingDeviceID         // device id is missing from the destination
	InconsistentDeviceID    // occurrences of device id in the event is inconsistent
	InvalidBootTime         // boot-time is either too far in the past or too far in the future
	MissingBootTime         // boot-time does not exist in the event's metadata
	OldBootTime             // boot-time is suspiciously old but not old enough to be deemed invalid
	NewerBootTimeFound      // event does not have the newest boot-time and therefore is an old event
	InvalidBootDuration     // default tag when event destination's unix timestamps are not in proper time range of event boot-time
	FastBoot                // event destination's unix timestamps are too close to the boot-time of the event
	InvalidBirthdate        // birthdate does not fall within a certain time range
	MisalignedBirthdate     // birthdate is not within a certain time range of the timestamps in the event destination
	InvalidDestination      // default tag when there is something wrong with the event destination
	NonEvent                // not an event
	InvalidEventType        // event type is not one of the possible event types
	EventTypeMismatch       // event type does not match what is being searched for
	DuplicateEvent          // duplicate event detected
	InconsistentMetadata    // metadata values for certain metadata keys are inconsistent
	RepeatedTransactionUUID // multiple events in an event list have the same transcation uuid
	MissingOnlineEvent      // session is missing online event
	MissingOfflineEvent     // session is missing offline event
	InvalidEventOrder       // wrong event order
	FalseReboot             // not a true reboot
	NoReboot                // no reboot found
)

const (
	UnknownStr                 = "unknown"
	PassStr                    = "pass"
	MultipleTagsStr            = "multiple_tags"
	MissingDeviceIDStr         = "missing_device_id"
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
	InvalidEventOrderStr       = "invalid_event_order"
	FalseRebootStr             = "false_reboot"
	NoRebootStr                = "no_reboot"
)

var (
	tagToString = map[Tag]string{
		Unknown:                 UnknownStr,
		Pass:                    PassStr,
		MultipleTags:            MultipleTagsStr,
		MissingDeviceID:         MissingDeviceIDStr,
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
		InvalidEventOrder:       InvalidEventOrderStr,
		FalseReboot:             FalseRebootStr,
		NoReboot:                NoRebootStr,
	}

	stringToTag = map[string]Tag{
		UnknownStr:                 Unknown,
		PassStr:                    Pass,
		MultipleTagsStr:            MultipleTags,
		MissingDeviceIDStr:         MissingDeviceID,
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
		InvalidEventOrderStr:       InvalidEventOrder,
		FalseRebootStr:             FalseReboot,
		NoRebootStr:                NoReboot,
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
