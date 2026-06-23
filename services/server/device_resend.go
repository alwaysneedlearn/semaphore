package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/semaphoreui/semaphore/db"
)

// ResendRangeInput is UI/API payload for manual resend (not from device profile config).
type ResendRangeInput struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Full  bool   `json:"full,omitempty"`
}

// ResendParams is playbook extra-var: type-specific formatted times + display for operation log.
type ResendParams struct {
	ProfileKey string `json:"profile_key"`
	Start      string `json:"start"`
	End        string `json:"end"`
	Full       bool   `json:"full"`
	Display    string `json:"display"`
}

func parseResendInstant(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("time is required")
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	var lastErr error
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return t, nil
		} else {
			lastErr = err
		}
	}
	if lastErr != nil {
		return time.Time{}, fmt.Errorf("invalid time %q: %w", raw, lastErr)
	}
	return time.Time{}, fmt.Errorf("invalid time %q", raw)
}

func formatLandLikeDateTime(t time.Time) string {
	return fmt.Sprintf("%d-%d-%d %d:%02d:%02d",
		t.Year(), int(t.Month()), t.Day(), t.Hour(), t.Minute(), t.Second())
}

func formatNBTDate(t time.Time) string {
	return fmt.Sprintf("%d-%d-%d", t.Year(), int(t.Month()), t.Day())
}

func formatNewareDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// BuildResendParams validates UI input and formats per device profile_key for Ansible.
func BuildResendParams(profileKey string, in ResendRangeInput) (ResendParams, error) {
	key := strings.ToUpper(strings.TrimSpace(profileKey))
	if key == "" {
		return ResendParams{}, fmt.Errorf("device profile key is empty")
	}

	if in.Full {
		switch key {
		case "JHAI":
			return ResendParams{
				ProfileKey: key,
				Full:       true,
				Display:    "全量重传",
			}, nil
		default:
			return ResendParams{}, fmt.Errorf("full resend is only supported for JHAI")
		}
	}

	startT, err := parseResendInstant(in.Start)
	if err != nil {
		return ResendParams{}, fmt.Errorf("start: %w", err)
	}
	endT, err := parseResendInstant(in.End)
	if err != nil {
		return ResendParams{}, fmt.Errorf("end: %w", err)
	}
	if !endT.After(startT) && !endT.Equal(startT) {
		return ResendParams{}, fmt.Errorf("end time must be on or after start time")
	}

	var startFmt, endFmt, display string
	switch key {
	case "NBT":
		startFmt = formatNBTDate(startT)
		endFmt = formatNBTDate(endT)
		display = startFmt + " → " + endFmt
	case "NEWARE":
		startFmt = formatNewareDate(startT)
		endFmt = formatNewareDate(endT)
		display = startFmt + " → " + endFmt
	case "LAND", "SINEXCEL", "JHAI":
		startFmt = formatLandLikeDateTime(startT)
		endFmt = formatLandLikeDateTime(endT)
		display = startFmt + " → " + endFmt
	default:
		return ResendParams{}, fmt.Errorf("resend is not supported for profile %q", key)
	}

	return ResendParams{
		ProfileKey: key,
		Start:      startFmt,
		End:        endFmt,
		Full:       false,
		Display:    display,
	}, nil
}

// MergeResendParamsExtraVars sets resend_params for playbooks (replaces profile/device config for time range).
func MergeResendParamsExtraVars(extraVars map[string]any, params ResendParams) {
	extraVars["resend_params"] = map[string]any{
		"profile_key": params.ProfileKey,
		"start":       params.Start,
		"end":         params.End,
		"full":        params.Full,
		"display":     params.Display,
	}
}

// ResendFormatHint returns a short UI hint for the profile's expected time precision.
func ResendFormatHint(profileKey string) string {
	switch strings.ToUpper(strings.TrimSpace(profileKey)) {
	case "NBT":
		return "Date only (yyyy-M-d); time picker uses the date part"
	case "NEWARE":
		return "Date (yyyy-MM-dd) written to HisDataFromTime / HisDataToTime"
	case "LAND":
		return "yyyy-M-d HH:mm:ss (e.g. 2026-6-1 10:10:10)"
	case "SINEXCEL", "JHAI":
		return "yyyy-M-d HH:mm:ss"
	default:
		return ""
	}
}

// ProfileSupportsResend reports whether profile_key has a resend_data implementation.
func ProfileSupportsResend(profileKey string) bool {
	switch strings.ToUpper(strings.TrimSpace(profileKey)) {
	case "JHAI", "LAND", "SINEXCEL", "NBT", "NEWARE":
		return true
	default:
		return false
	}
}

// ValidateResendForProfile ensures profile can resend before template lookup.
func ValidateResendForProfile(profile db.DeviceProfile) error {
	if !ProfileSupportsResend(profile.ProfileKey) {
		return db.NewValidationError(fmt.Sprintf("Device profile %q does not support resend data", profile.ProfileKey))
	}
	return nil
}
