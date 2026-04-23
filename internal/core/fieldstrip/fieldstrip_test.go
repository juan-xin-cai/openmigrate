package fieldstrip

import (
	"strings"
	"testing"

	"github.com/openmigrate/openmigrate/internal/core/types"
)

func TestStripRemovesPrefixExactAndGlobRules(t *testing.T) {
	data := []byte(`{
	  "oauth:token": "abc",
	  "keep": "ok",
	  "electron": {
	    "media": {
	      "device_id_salt": "salt",
	      "other": "keep"
	    }
	  },
	  "partition": {
	    "one": {
	      "per_host_zoom_levels": {"a": 1},
	      "keep": true
	    },
	    "two": {
	      "per_host_zoom_levels": {"b": 2}
	    }
	  }
	}`)
	clean, err := Strip(data, []types.FieldStripRule{
		{Type: types.FieldStripRulePrefix, Value: "oauth:"},
		{Type: types.FieldStripRuleExactPath, Value: "electron.media.device_id_salt"},
		{Type: types.FieldStripRuleGlobPath, Value: "partition.*.per_host_zoom_levels"},
	})
	if err != nil {
		t.Fatalf("strip: %v", err)
	}
	text := string(clean)
	for _, needle := range []string{"oauth:token", "device_id_salt", "per_host_zoom_levels"} {
		if strings.Contains(text, needle) {
			t.Fatalf("unexpected %q in %s", needle, text)
		}
	}
	if !strings.Contains(text, `"keep": "ok"`) || !strings.Contains(text, `"other": "keep"`) {
		t.Fatalf("expected untouched keys in %s", text)
	}
}

func TestStripReturnsErrNotJSON(t *testing.T) {
	_, err := Strip([]byte("not-json"), []types.FieldStripRule{{Type: types.FieldStripRulePrefix, Value: "oauth:"}})
	if err != types.ErrNotJSON {
		t.Fatalf("err = %v", err)
	}
}

func TestStripNoRulesReturnsCopy(t *testing.T) {
	data := []byte(`{"keep":true}`)
	clean, err := Strip(data, nil)
	if err != nil {
		t.Fatalf("strip: %v", err)
	}
	if string(clean) != string(data) {
		t.Fatalf("clean = %s", clean)
	}
	if len(clean) > 0 {
		clean[0] = '['
	}
	if string(data) != `{"keep":true}` {
		t.Fatalf("original mutated: %s", data)
	}
}
