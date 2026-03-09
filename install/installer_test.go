package install

import (
	"testing"
)

func TestHasCtxHook_Found(t *testing.T) {
	hooks := map[string]interface{}{
		"PreCompact": []interface{}{
			map[string]interface{}{
				"matcher": "auto",
				"hooks": []interface{}{
					map[string]interface{}{"type": "command", "command": "ctx hook precompact"},
				},
			},
		},
	}

	if !hasCtxHook(hooks, "PreCompact", precompactCmd) {
		t.Error("expected hasCtxHook to find PreCompact hook")
	}
}

func TestHasCtxHook_Missing(t *testing.T) {
	hooks := map[string]interface{}{
		"PreCompact": []interface{}{
			map[string]interface{}{
				"matcher": "auto",
				"hooks": []interface{}{
					map[string]interface{}{"type": "command", "command": "some-other-tool"},
				},
			},
		},
	}

	if hasCtxHook(hooks, "PreCompact", precompactCmd) {
		t.Error("expected hasCtxHook to not find ctx command")
	}
}

func TestHasCtxHook_KeyMissing(t *testing.T) {
	hooks := map[string]interface{}{}

	if hasCtxHook(hooks, "PreCompact", precompactCmd) {
		t.Error("expected hasCtxHook to return false for missing key")
	}
}

func TestHasCtxHook_SessionStart(t *testing.T) {
	hooks := map[string]interface{}{
		"SessionStart": []interface{}{
			map[string]interface{}{
				"hooks": []interface{}{
					map[string]interface{}{"type": "command", "command": "ctx hook session"},
				},
			},
		},
	}

	if !hasCtxHook(hooks, "SessionStart", sessionCmd) {
		t.Error("expected hasCtxHook to find SessionStart hook")
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   bool
	}{
		{"exact match", "hello", "hello", true},
		{"substring at start", "hello world", "hello", true},
		{"substring at end", "hello world", "world", true},
		{"substring in middle", "hello world foo", "world", true},
		{"not found", "hello", "xyz", false},
		{"empty substr", "hello", "", true},
		{"empty both", "", "", true},
		{"empty s non-empty substr", "", "a", false},
		{"substr longer than s", "hi", "hello", false},
		{"single char found", "abc", "b", true},
		{"single char not found", "abc", "z", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestSearchString(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   bool
	}{
		{"found at start", "abcdef", "abc", true},
		{"found at end", "abcdef", "def", true},
		{"found in middle", "abcdef", "cde", true},
		{"not found", "abcdef", "xyz", false},
		{"empty substr", "abc", "", true},
		{"same length match", "abc", "abc", true},
		{"same length no match", "abc", "xyz", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := searchString(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("searchString(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestGetOrCreateHooksMap_NoHooksKey(t *testing.T) {
	settings := Settings{}
	hooks := getOrCreateHooksMap(settings)
	if hooks == nil {
		t.Fatal("expected non-nil map")
	}
	if len(hooks) != 0 {
		t.Errorf("expected empty map, got %d entries", len(hooks))
	}
}

func TestGetOrCreateHooksMap_InvalidType(t *testing.T) {
	settings := Settings{"hooks": "not-a-map"}
	hooks := getOrCreateHooksMap(settings)
	if hooks == nil {
		t.Fatal("expected non-nil map")
	}
	if len(hooks) != 0 {
		t.Errorf("expected empty map, got %d entries", len(hooks))
	}
}

func TestGetOrCreateHooksMap_ValidMap(t *testing.T) {
	inner := map[string]interface{}{"PreCompact": []interface{}{}}
	settings := Settings{"hooks": inner}
	hooks := getOrCreateHooksMap(settings)
	if _, ok := hooks["PreCompact"]; !ok {
		t.Error("expected PreCompact key in returned map")
	}
}
