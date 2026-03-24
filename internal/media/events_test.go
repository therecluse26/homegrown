package media

import "testing"

func TestUploadPublished_EventName(t *testing.T) {
	e := UploadPublished{}
	if e.EventName() != "media.UploadPublished" {
		t.Errorf("EventName() = %q, want %q", e.EventName(), "media.UploadPublished")
	}
}

func TestUploadQuarantined_EventName(t *testing.T) {
	e := UploadQuarantined{}
	if e.EventName() != "media.UploadQuarantined" {
		t.Errorf("EventName() = %q, want %q", e.EventName(), "media.UploadQuarantined")
	}
}

func TestUploadRejected_EventName(t *testing.T) {
	e := UploadRejected{}
	if e.EventName() != "media.UploadRejected" {
		t.Errorf("EventName() = %q, want %q", e.EventName(), "media.UploadRejected")
	}
}

func TestUploadFlagged_EventName(t *testing.T) {
	e := UploadFlagged{}
	if e.EventName() != "media.UploadFlagged" {
		t.Errorf("EventName() = %q, want %q", e.EventName(), "media.UploadFlagged")
	}
}
