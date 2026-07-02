package dm

import "errors"

const (
	PrivacyEveryone    = "everyone"
	PrivacyFriendsOnly = "friends_only"
	PrivacyOff         = "off"
)

var (
	ErrSelfDM         = errors.New("cannot message yourself")
	ErrDMDisabled     = errors.New("this member is not accepting messages")
	ErrDMFriendsOnly  = errors.New("this member only accepts messages from existing contacts")
	ErrSenderDMOff    = errors.New("enable messages in your privacy settings to send DMs")
	ErrNotParticipant = errors.New("not a participant in this conversation")
)

func NormalizePrivacy(v string) string {
	switch v {
	case PrivacyFriendsOnly, PrivacyOff:
		return v
	default:
		return PrivacyEveryone
	}
}

func ValidPrivacy(v string) bool {
	return v == PrivacyEveryone || v == PrivacyFriendsOnly || v == PrivacyOff
}