package rules

import (
	"strconv"
	"strings"
)

// ParseTaxonomy is a deliberately small YAML reader for the backward-compatible
// assessment_rules block. The source-of-truth taxonomy remains ordinary YAML;
// the generator only needs this declarative extension and therefore avoids a
// runtime dependency on a full YAML stack.
func ParseTaxonomy(b []byte) Taxonomy {
	lines := strings.Split(string(b), "\n")
	var rules []Rule
	inRules := false
	var cur *Rule
	for _, raw := range lines {
		line := strings.TrimRight(raw, " \t")
		trimmed := strings.TrimSpace(line)
		if trimmed == "assessment_rules:" {
			inRules = true
			continue
		}
		if !inRules || trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(line, " ") && strings.HasSuffix(trimmed, ":") {
			if cur != nil {
				rules = append(rules, *cur)
				cur = nil
			}
			inRules = false
			continue
		}
		if strings.HasPrefix(trimmed, "- rule_id:") {
			if cur != nil {
				rules = append(rules, *cur)
			}
			cur = &Rule{RuleID: unquote(after(trimmed, "- rule_id:"))}
			continue
		}
		if cur == nil {
			continue
		}
		switch {
		case strings.HasPrefix(trimmed, "test_id:"):
			cur.TestID, _ = strconv.Atoi(strings.TrimSpace(after(trimmed, "test_id:")))
		case strings.HasPrefix(trimmed, "result_text:"):
			cur.ResultText = unquote(after(trimmed, "result_text:"))
		case strings.HasPrefix(trimmed, "strength:"):
			cur.Strength = unquote(after(trimmed, "strength:"))
		case strings.HasPrefix(trimmed, "equals:"):
			val := unquote(after(trimmed, "equals:"))
			// The handoff rules use one fact equality per rule. Keep this small
			// parser deterministic while the evaluator supports richer predicates
			// for tests and future programmatic Taxonomy construction.
			if len(cur.When.All) == 0 || cur.When.All[len(cur.When.All)-1].Equals != nil {
				cur.When.All = append(cur.When.All, Condition{})
			}
			cur.When.All[len(cur.When.All)-1].Equals = val
		case strings.HasPrefix(trimmed, "fact:") || strings.HasPrefix(trimmed, "- fact:"):
			fact := unquote(after(strings.TrimPrefix(trimmed, "- "), "fact:"))
			if len(cur.When.All) == 0 || cur.When.All[len(cur.When.All)-1].Fact != "" {
				cur.When.All = append(cur.When.All, Condition{})
			}
			cur.When.All[len(cur.When.All)-1].Fact = fact
		}
	}
	if cur != nil {
		rules = append(rules, *cur)
	}
	return Taxonomy{AssessmentRules: rules}
}
func after(s, p string) string { return strings.TrimSpace(strings.TrimPrefix(s, p)) }
func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		s = s[1 : len(s)-1]
	}
	s = strings.ReplaceAll(s, `\"`, `"`)
	s = strings.ReplaceAll(s, `\\`, `\`)
	return s
}
