package fieldstrip

import (
	"encoding/json"
	"strings"

	"github.com/openmigrate/openmigrate/internal/core/types"
)

func Strip(data []byte, rules []types.FieldStripRule) ([]byte, error) {
	if len(rules) == 0 {
		return append([]byte(nil), data...), nil
	}

	var payload interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, types.ErrNotJSON
	}

	for _, rule := range rules {
		applyRule(payload, rule)
	}

	clean, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, err
	}
	return clean, nil
}

func applyRule(node interface{}, rule types.FieldStripRule) {
	switch rule.Type {
	case types.FieldStripRulePrefix:
		root, ok := node.(map[string]interface{})
		if !ok {
			return
		}
		for key := range root {
			if strings.HasPrefix(key, rule.Value) {
				delete(root, key)
			}
		}
	case types.FieldStripRuleExactPath:
		deletePath(node, strings.Split(rule.Value, "."), false)
	case types.FieldStripRuleGlobPath:
		deletePath(node, strings.Split(rule.Value, "."), true)
	}
}

func deletePath(node interface{}, segments []string, allowGlob bool) {
	if len(segments) == 0 {
		return
	}
	object, ok := node.(map[string]interface{})
	if !ok {
		return
	}
	head := segments[0]
	if len(segments) == 1 {
		if head == "*" && allowGlob {
			for key := range object {
				delete(object, key)
			}
			return
		}
		delete(object, head)
		return
	}
	if head == "*" && allowGlob {
		for _, child := range object {
			deletePath(child, segments[1:], allowGlob)
		}
		return
	}
	child, ok := object[head]
	if !ok {
		return
	}
	deletePath(child, segments[1:], allowGlob)
}
