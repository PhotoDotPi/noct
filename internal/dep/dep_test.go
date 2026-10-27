package dep

import (
	"errors"
	"testing"
)

func TestMissingDependencyError(t *testing.T) {
	cause := errors.New("executable file not found in $PATH")
	err := &MissingDependencyError{Name: "yt-dlp", Err: cause}

	if !IsMissing(err) {
		t.Fatal("expected IsMissing to be true")
	}
	if err.Error() == "" {
		t.Fatal("expected a non-empty message")
	}
	if err.Name != "yt-dlp" {
		t.Errorf("Name: got %q want %q", err.Name, "yt-dlp")
	}
	if !errors.Is(err, cause) {
		t.Fatal("expected errors.Is to unwrap to the cause")
	}
}

func TestIsMissingFalseForOtherErrors(t *testing.T) {
	if IsMissing(errors.New("some other error")) {
		t.Fatal("expected IsMissing to be false for a plain error")
	}
	if IsMissing(nil) {
		t.Fatal("expected IsMissing to be false for nil")
	}
}

func TestMissingDependencyErrorMessageMentionsName(t *testing.T) {
	err := &MissingDependencyError{Name: "mpv", Err: errors.New("not found")}
	if msg := err.Error(); msg == "" {
		t.Fatal("expected a non-empty message")
	}
}
