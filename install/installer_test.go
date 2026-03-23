package install

import (
	"testing"
)

func TestHasCtxHook_Found(t *testing.T) {
	cmd := "ctx hook precompact --trigger=auto"
	hooks := map[string]interface{}{
		"PreCompact": []interface{}{
			map[string]interface{}{
				"matcher": "auto",
				"hooks": []interface{}{
					map[string]interface{}{"type": "command", "command": cmd},
				},
			},
		},
	}

	if !hasCtxHook(hooks, "PreCompact", cmd) {
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

	if hasCtxHook(hooks, "PreCompact", "ctx hook precompact") {
		t.Error("expected hasCtxHook to not find ctx command")
	}
}

func TestHasCtxHook_KeyMissing(t *testing.T) {
	hooks := map[string]interface{}{}

	if hasCtxHook(hooks, "PreCompact", "ctx hook precompact") {
		t.Error("expected hasCtxHook to return false for missing key")
	}
}

func TestHasCtxHook_SessionStart(t *testing.T) {
	cmd := "ctx hook session"
	hooks := map[string]interface{}{
		"SessionStart": []interface{}{
			map[string]interface{}{
				"hooks": []interface{}{
					map[string]interface{}{"type": "command", "command": cmd},
				},
			},
		},
	}

	if !hasCtxHook(hooks, "SessionStart", cmd) {
		t.Error("expected hasCtxHook to find SessionStart hook")
	}
}

func TestHasCtxHook_StringMatch(t *testing.T) {
	hooks := map[string]interface{}{
		"PreCompact": []interface{}{
			map[string]interface{}{
				"hooks": []interface{}{
					map[string]interface{}{"type": "command", "command": "/usr/local/bin/ctx hook precompact"},
				},
			},
		},
	}
	if !hasCtxHook(hooks, "PreCompact", "hook precompact") {
		t.Error("expected hasCtxHook to return true for matching command")
	}
	if hasCtxHook(hooks, "PreCompact", "hook session") {
		t.Error("expected hasCtxHook to return false for non-matching command")
	}
	if hasCtxHook(hooks, "SessionStart", "hook session") {
		t.Error("expected hasCtxHook to return false for missing key")
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
