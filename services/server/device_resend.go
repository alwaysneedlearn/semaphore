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
	// Full is rejected; kept only so old clients get a clear error instead of silent ignore.
	Full bool `json:"full,omitempty"`
}

// ResendParams is playbook extra-var. Start/End are canonical local times
// ("2006-01-02 15:04:05"); each playbook formats for its vendor API.
type ResendParams struct {
	ProfileKey string `json:"profile_key"`
	Start      string `json:"start"`
	End        string `json:"end"`
	Display    string `json:"display"`
}

// CanonicalResendTimeLayout is the fixed format Semaphore puts in resend_params;
// cursor-playbooks/shared/tasks/semaphore_resend_apply_from_params.yml parses it.
const CanonicalResendTimeLayout = "2006-01-02 15:04:05"

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

// BuildResendParams validates UI input and emits canonical times for Ansible.
// Per-vendor string shaping happens in playbooks (semaphore_resend_apply_from_params.yml).
func BuildResendParams(profileKey string, in ResendRangeInput) (ResendParams, error) {
	key := strings.ToUpper(strings.TrimSpace(profileKey))
	if key == "" {
		return ResendParams{}, fmt.Errorf("device profile key is empty")
	}
	if in.Full {
		return ResendParams{}, fmt.Errorf("full resend is not supported; provide start and end")
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

	startFmt := startT.Format(CanonicalResendTimeLayout)
	endFmt := endT.Format(CanonicalResendTimeLayout)
	return ResendParams{
		ProfileKey: key,
		Start:      startFmt,
		End:        endFmt,
		Display:    startFmt + " → " + endFmt,
	}, nil
}

// MergeResendParamsExtraVars sets resend_params for playbooks (replaces profile/device config for time range).
func MergeResendParamsExtraVars(extraVars map[string]any, params ResendParams) {
	extraVars["resend_params"] = map[string]any{
		"profile_key": params.ProfileKey,
		"start":       params.Start,
		"end":         params.End,
		"display":     params.Display,
	}
}

// ValidateResendTemplateBound ensures the profile has a resend_data template bound.
func ValidateResendTemplateBound(ps db.ProjectDeviceProfileSettings) error {
	if ps.ResendDataTemplateID == nil || *ps.ResendDataTemplateID == 0 {
		return db.NewValidationError("No resend data template configured for this device profile")
	}
	return nil
}
