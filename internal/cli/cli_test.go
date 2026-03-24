package cli

import (
	"testing"
)

func TestSplitArgs_Simple(t *testing.T) {
	got := splitArgs("/invoke browser --action snapshot")
	want := []string{"/invoke", "browser", "--action", "snapshot"}
	assertSlice(t, got, want)
}

func TestSplitArgs_QuotedJSON(t *testing.T) {
	got := splitArgs(`/invoke browser --args '{"url":"https://example.com"}'`)
	want := []string{"/invoke", "browser", "--args", `{"url":"https://example.com"}`}
	assertSlice(t, got, want)
}

func TestSplitArgs_Empty(t *testing.T) {
	got := splitArgs("")
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func TestSplitArgs_ExtraSpaces(t *testing.T) {
	got := splitArgs("  /help  ")
	want := []string{"/help"}
	assertSlice(t, got, want)
}

func TestSplitArgs_ChatMessage(t *testing.T) {
	got := splitArgs("Hello, how are you?")
	want := []string{"Hello,", "how", "are", "you?"}
	assertSlice(t, got, want)
}

func TestFlagValue_Found(t *testing.T) {
	args := []string{"browser", "--action", "snapshot", "--instance", "prod"}
	if v := flagValue(args, "--action", "-a"); v != "snapshot" {
		t.Errorf("action = %q", v)
	}
	if v := flagValue(args, "--instance", "-i"); v != "prod" {
		t.Errorf("instance = %q", v)
	}
}

func TestFlagValue_ShortFlag(t *testing.T) {
	args := []string{"browser", "-a", "snapshot"}
	if v := flagValue(args, "--action", "-a"); v != "snapshot" {
		t.Errorf("action = %q", v)
	}
}

func TestFlagValue_NotFound(t *testing.T) {
	args := []string{"browser", "--action", "snapshot"}
	if v := flagValue(args, "--instance"); v != "" {
		t.Errorf("expected empty, got %q", v)
	}
}

func TestFlagValue_AtEnd(t *testing.T) {
	args := []string{"browser", "--action"}
	if v := flagValue(args, "--action"); v != "" {
		t.Errorf("expected empty for flag at end, got %q", v)
	}
}

func assertSlice(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %v", len(got), len(want), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
