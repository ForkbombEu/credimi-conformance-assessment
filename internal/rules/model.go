package rules

type Taxonomy struct {
	AssessmentRules    []Rule         `yaml:"assessment_rules"`
	NormalizationRules map[string]any `yaml:"normalization_rules"`
}
type Rule struct {
	RuleID     string    `yaml:"rule_id"`
	TestID     int       `yaml:"test_id"`
	ResultText string    `yaml:"result_text"`
	Strength   string    `yaml:"strength"`
	When       Condition `yaml:"when"`
}
type Condition struct {
	All          []Condition `yaml:"all"`
	Any          []Condition `yaml:"any"`
	Not          *Condition  `yaml:"not"`
	Fact         string      `yaml:"fact"`
	Field        string      `yaml:"field"`
	Equals       any         `yaml:"equals"`
	NotEquals    any         `yaml:"not_equals"`
	Contains     any         `yaml:"contains"`
	ContainsAny  []any       `yaml:"contains_any"`
	Exists       *bool       `yaml:"exists"`
	MatchesRegex string      `yaml:"matches_regex"`
	LTE          any         `yaml:"lte"`
	GTE          any         `yaml:"gte"`
}
type Result struct {
	TestID   int
	Text     string
	RuleID   string
	Strength string
}
