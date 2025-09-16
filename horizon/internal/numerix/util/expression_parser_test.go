package util

import (
	"testing"
)

func TestIsOp(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected bool
	}{
		// Binary operators
		{"Addition", "+", true},
		{"Subtraction", "-", true},
		{"Multiplication", "*", true},
		{"Division", "/", true},
		{"Power", "^", true},
		{"Greater than", ">", true},
		{"Less than", "<", true},
		{"Greater than or equal", ">=", true},
		{"Less than or equal", "<=", true},
		{"Equal", "==", true},
		{"Min", "min", true},
		{"Max", "max", true},
		{"Logical AND", "&", true},
		{"Logical OR", "|", true},

		// Unary operators
		{"Exponential", "exp", true},
		{"Logarithm", "log", true},
		{"Absolute", "abs", true},
		{"Normalize min max", "norm_min_max", true},
		{"Percentile rank", "percentile_rank", true},
		{"Normalize percentile 0-99", "norm_percentile_0_99", true},
		{"Normalize percentile 5-95", "norm_percentile_5_95", true},

		// Non-operators
		{"Variable", "variable", false},
		{"Number", "123", false},
		{"Decimal", "12.34", false},
		{"Parenthesis open", "(", false},
		{"Parenthesis close", ")", false},
		{"Comma", ",", false},
		{"Empty string", "", false},
		{"Space", " ", false},
		{"Invalid operator", "++", false},
		{"Case sensitive", "EXP", false},
		{"Mixed case", "Min", false},
		{"Partial match", "mi", false},
		{"Extra characters", "min2", false},
		{"Special characters", "@", false},
		{"Unicode", "λ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsOp(tt.token)
			if result != tt.expected {
				t.Errorf("IsOp(%s) = %v, expected %v", tt.token, result, tt.expected)
			}
		})
	}
}

func TestIsNumber(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected bool
	}{
		// Valid numbers
		{"Positive integer", "123", true},
		{"Negative integer", "-123", true},
		{"Zero", "0", true},
		{"Decimal", "12.34", true},
		{"Negative decimal", "-12.34", true},
		{"Scientific notation", "1.23e10", true},
		{"Scientific notation negative", "-1.23e-10", true},
		{"Leading zero", "0123", true},
		{"Decimal with leading zero", "0.123", true},
		{"Just decimal point", ".123", true},
		{"Decimal ending with zero", "123.0", true},
		{"Large number", "999999999999999999", true},
		{"Very small decimal", "0.000000001", true},
		{"Scientific E upper", "1.23E10", true},
		{"Scientific with plus", "1.23e+10", true},

		// Invalid numbers
		{"Empty string", "", false},
		{"Just decimal point", ".", false},
		{"Multiple decimal points", "12.34.56", false},
		{"Letters", "abc", false},
		{"Mixed letters and numbers", "12abc", false},
		{"Space", " ", false},
		{"Number with space", "12 34", false},
		{"Leading space", " 123", false},
		{"Trailing space", "123 ", false},
		{"Operator", "+", false},
		{"Variable", "variable", false},
		{"Parenthesis", "(", false},
		{"Comma", ",", false},
		{"Double negative", "--123", false},
		{"Plus sign", "+123", true},
		{"Scientific notation invalid", "1.23e", false},
		{"Scientific notation double e", "1.23e2e3", false},
		{"Hex number", "0x123", false},
		{"Binary number", "0b101", false},
		{"Infinity", "inf", true},
		{"NaN", "nan", true},
		{"Unicode numbers", "١٢٣", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNumber(tt.token)
			if result != tt.expected {
				t.Errorf("IsNumber(%s) = %v, expected %v", tt.token, result, tt.expected)
			}
		})
	}
}

func TestExtractVariables(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		expected   []string
	}{
		// Single variable
		{"Single variable", "x", []string{"x"}},
		{"Variable with underscores", "var_name", []string{"var_name"}},
		{"Variable with numbers", "var123", []string{"var123"}},

		// Multiple variables
		{"Two variables", "x y", []string{"x", "y"}},
		{"Variables with operators", "x + y", []string{"x", "y"}},
		{"Variables with functions", "exp x", []string{"x"}},
		{"Variables with parentheses", "( x + y )", []string{"(", "x", "y", ")"}},
		{"Complex expression", "x + y * z", []string{"x", "y", "z"}},

		// Variables with numbers and operators
		{"Mixed tokens", "x + 123 - y", []string{"x", "y"}},
		{"With decimal numbers", "x + 12.34 * y", []string{"x", "y"}},
		{"With negative numbers", "x + -123 * y", []string{"x", "y"}},
		{"With scientific notation", "x + 1.23e10 * y", []string{"x", "y"}},

		// Variables with unary operators
		{"With exp function", "exp x + y", []string{"x", "y"}},
		{"With log function", "log x + log y", []string{"x", "y"}},
		{"With abs function", "abs x", []string{"x"}},
		{"With norm functions", "norm_min_max x", []string{"x"}},
		{"With percentile functions", "percentile_rank x", []string{"x"}},

		// Variables with binary operators
		{"With min/max", "min x y", []string{"x", "y"}},
		{"With comparison", "x > y", []string{"x", "y"}},
		{"With logical operators", "x & y", []string{"x", "y"}},
		{"With all operators", "x + y - z * w / v ^ u", []string{"x", "y", "z", "w", "v", "u"}},

		// Edge cases
		{"Empty expression", "", []string{}},
		{"Only numbers", "123 456", []string{}},
		{"Only operators", "+ - * /", []string{}},
		{"Only parentheses", "( ) ( )", []string{"(", ")", "(", ")"}},
		{"Numbers and operators", "123 + 456 - 789", []string{}},
		{"Function calls", "exp log abs", []string{}},
		{"Mixed parentheses", "( 123 + 456 )", []string{"(", ")"}},

		// Duplicate variables (should be preserved as is)
		{"Duplicate variables", "x + x", []string{"x", "x"}},
		{"Multiple duplicates", "x + y + x + y", []string{"x", "y", "x", "y"}},

		// Special variable names
		{"Underscore variable", "_var", []string{"_var"}},
		{"Leading underscore", "_", []string{"_"}},
		{"Camel case", "camelCase", []string{"camelCase"}},
		{"Snake case", "snake_case_var", []string{"snake_case_var"}},
		{"Long variable name", "very_long_variable_name_123", []string{"very_long_variable_name_123"}},

		// Complex realistic expressions
		{"Realistic expression 1", "score + weight * factor", []string{"score", "weight", "factor"}},
		{"Realistic expression 2", "exp ( log price + discount_rate )", []string{"(", "price", "discount_rate", ")"}},
		{"Realistic expression 3", "min rating quality_score", []string{"rating", "quality_score"}},
		{"Realistic expression 4", "norm_min_max engagement_score", []string{"engagement_score"}},
		{"Realistic expression 5", "( base_score + bonus ) / total_items", []string{"(", "base_score", "bonus", ")", "total_items"}},

		// Edge cases with spacing
		{"No spaces", "x+y*z", []string{"x+y*z"}},
		{"Extra spaces", "  x   +   y   *   z  ", []string{"x", "y", "z"}},
		{"Mixed spacing", "x+ y * z", []string{"x+", "y", "z"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractVariables(tt.expression)
			if !equalSlices(result, tt.expected) {
				t.Errorf("ExtractVariables(%s) = %v, expected %v", tt.expression, result, tt.expected)
			}
		})
	}
}

// Helper function to compare string slices
func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
