package server

import (
	"testing"

	"github.com/semaphoreui/semaphore/db"
)

func TestBuildResendParamsCanonical(t *testing.T) {
	params, err := BuildResendParams("LANDV7", ResendRangeInput{
		Start: "2026-06-01T10:10:10",
		End:   "2026-06-02T11:20:30",
	})
	if err != nil {
		t.Fatal(err)
	}
	if params.Start != "2026-06-01 10:10:10" {
		t.Fatalf("start=%q want canonical", params.Start)
	}
	if params.End != "2026-06-02 11:20:30" {
		t.Fatalf("end=%q want canonical", params.End)
	}
	if params.ProfileKey != "LANDV7" {
		t.Fatalf("key=%q", params.ProfileKey)
	}
	if params.Display == "" {
		t.Fatal("display empty")
	}
}

func TestBuildResendParamsAnyProfileKey(t *testing.T) {
	params, err := BuildResendParams("CUSTOMTYPE", ResendRangeInput{
		Start: "2026-06-01 08:00:00",
		End:   "2026-06-03 08:00:00",
	})
	if err != nil {
		t.Fatal(err)
	}
	if params.Start != "2026-06-01 08:00:00" || params.End != "2026-06-03 08:00:00" {
		t.Fatalf("got %q → %q", params.Start, params.End)
	}
}

func TestBuildResendParamsRejectsFull(t *testing.T) {
	_, err := BuildResendParams("JHAI", ResendRangeInput{Full: true})
	if err == nil {
		t.Fatal("expected error for full resend")
	}
}

func TestBuildResendParamsEndBeforeStart(t *testing.T) {
	_, err := BuildResendParams("LAND", ResendRangeInput{
		Start: "2026-06-02T12:00:00",
		End:   "2026-06-01T12:00:00",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateResendTemplateBound(t *testing.T) {
	if err := ValidateResendTemplateBound(db.ProjectDeviceProfileSettings{}); err == nil {
		t.Fatal("expected error when template unset")
	}
	id := 42
	if err := ValidateResendTemplateBound(db.ProjectDeviceProfileSettings{ResendDataTemplateID: &id}); err != nil {
		t.Fatal(err)
	}
}
