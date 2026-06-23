package server

import (
	"testing"
	"time"
)

func TestBuildResendParamsLAND(t *testing.T) {
	params, err := BuildResendParams("LAND", ResendRangeInput{
		Start: "2026-06-01T10:10:10",
		End:   "2026-06-02T11:20:30",
	})
	if err != nil {
		t.Fatal(err)
	}
	if params.Start != "2026-6-1 10:10:10" {
		t.Fatalf("start=%q", params.Start)
	}
	if params.End != "2026-6-2 11:20:30" {
		t.Fatalf("end=%q", params.End)
	}
	if params.Display == "" {
		t.Fatal("display empty")
	}
}

func TestBuildResendParamsNBT(t *testing.T) {
	params, err := BuildResendParams("NBT", ResendRangeInput{
		Start: "2026-06-01T08:00:00",
		End:   "2026-06-03T08:00:00",
	})
	if err != nil {
		t.Fatal(err)
	}
	if params.Start != "2026-6-1" || params.End != "2026-6-3" {
		t.Fatalf("got %q → %q", params.Start, params.End)
	}
}

func TestBuildResendParamsJHAIFull(t *testing.T) {
	params, err := BuildResendParams("JHAI", ResendRangeInput{Full: true})
	if err != nil {
		t.Fatal(err)
	}
	if !params.Full || params.Display != "全量重传" {
		t.Fatalf("%+v", params)
	}
}

func TestBuildResendParamsEndBeforeStart(t *testing.T) {
	_, err := BuildResendParams("LAND", ResendRangeInput{
		Start: "2026-06-02T10:00:00",
		End:   "2026-06-01T10:00:00",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseResendInstantDateOnly(t *testing.T) {
	tm, err := parseResendInstant("2026-06-01")
	if err != nil {
		t.Fatal(err)
	}
	if tm.Year() != 2026 || tm.Month() != time.June || tm.Day() != 1 {
		t.Fatalf("%v", tm)
	}
}
