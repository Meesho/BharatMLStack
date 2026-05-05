package util

import (
	"fmt"
	"strings"
	"testing"
)

func TestIsUnaryOperator(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected bool
	}{
		// Valid unary operators
		{"Exponential", "exp", true},
		{"Logarithm", "log", true},
		{"Absolute", "abs", true},
		{"Normalize min max", "norm_min_max", true},
		{"Percentile rank", "percentile_rank", true},
		{"Normalize percentile 0-99", "norm_percentile_0_99", true},
		{"Normalize percentile 5-95", "norm_percentile_5_95", true},
		{"Min function", "min", true},
		{"Max function", "max", true},

		// Invalid unary operators
		{"Addition", "+", false},
		{"Subtraction", "-", false},
		{"Multiplication", "*", false},
		{"Division", "/", false},
		{"Power", "^", false},
		{"Greater than", ">", false},
		{"Less than", "<", false},
		{"Greater than or equal", ">=", false},
		{"Less than or equal", "<=", false},
		{"Equal", "==", false},
		{"Logical AND", "&", false},
		{"Logical OR", "|", false},
		{"Variable", "variable", false},
		{"Number", "123", false},
		{"Empty string", "", false},
		{"Case sensitive", "EXP", false},
		{"Mixed case", "Exp", false},
		{"Partial match", "ex", false},
		{"Extra characters", "exp2", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsUnaryOperator(tt.token)
			if result != tt.expected {
				t.Errorf("IsUnaryOperator(%s) = %v, expected %v", tt.token, result, tt.expected)
			}
		})
	}
}

func TestIsOperator(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsOperator(tt.token)
			if result != tt.expected {
				t.Errorf("IsOperator(%s) = %v, expected %v", tt.token, result, tt.expected)
			}
		})
	}
}

func TestIsVariable(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected bool
	}{
		// Valid variables
		{"Simple variable", "x", true},
		{"Variable with underscores", "var_name", true},
		{"Variable with numbers", "var123", true},
		{"Camel case", "camelCase", true},
		{"Snake case", "snake_case_var", true},
		{"Underscore variable", "_var", true},
		{"Leading underscore", "_", true},
		{"Long variable name", "very_long_variable_name_123", true},
		{"Mixed case", "VarName", true},

		// Invalid variables (numbers)
		{"Integer", "123", false},
		{"Decimal", "12.34", false},
		{"Negative number", "-123", false},
		{"Scientific notation", "1.23e10", false},
		{"Zero", "0", false},

		// Invalid variables (punctuation)
		{"Open parenthesis", "(", false},
		{"Close parenthesis", ")", false},
		{"Comma", ",", false},

		// Invalid variables (operators)
		{"Addition", "+", false},
		{"Subtraction", "-", false},
		{"Multiplication", "*", false},
		{"Division", "/", false},
		{"Power", "^", false},
		{"Greater than", ">", false},
		{"Less than", "<", false},
		{"Greater than or equal", ">=", false},
		{"Less than or equal", "<=", false},
		{"Equal", "==", false},
		{"Logical AND", "&", false},
		{"Logical OR", "|", false},
		{"Min", "min", false},
		{"Max", "max", false},
		{"Exponential", "exp", false},
		{"Logarithm", "log", false},
		{"Absolute", "abs", false},
		{"Normalize min max", "norm_min_max", false},
		{"Percentile rank", "percentile_rank", false},
		{"Normalize percentile 0-99", "norm_percentile_0_99", false},
		{"Normalize percentile 5-95", "norm_percentile_5_95", false},

		// Edge cases
		{"Empty string", "", true},
		{"Space", " ", true},
		{"Special characters", "@", true},
		{"Unicode", "λ", true},
		{"Number with letters", "12abc", true}, // This is actually considered a variable by the current logic
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isVariable(tt.token)
			if result != tt.expected {
				t.Errorf("isVariable(%s) = %v, expected %v", tt.token, result, tt.expected)
			}
		})
	}
}

func TestIsBinaryOperator(t *testing.T) {
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
		{"Logical AND", "&", true},
		{"Logical OR", "|", true},

		// Non-binary operators
		{"Min", "min", false},
		{"Max", "max", false},
		{"Exponential", "exp", false},
		{"Logarithm", "log", false},
		{"Absolute", "abs", false},
		{"Normalize min max", "norm_min_max", false},
		{"Percentile rank", "percentile_rank", false},
		{"Normalize percentile 0-99", "norm_percentile_0_99", false},
		{"Normalize percentile 5-95", "norm_percentile_5_95", false},
		{"Variable", "variable", false},
		{"Number", "123", false},
		{"Parenthesis", "(", false},
		{"Comma", ",", false},
		{"Empty string", "", false},
		{"Invalid operator", "++", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBinaryOperator(tt.token)
			if result != tt.expected {
				t.Errorf("IsBinaryOperator(%s) = %v, expected %v", tt.token, result, tt.expected)
			}
		})
	}
}

func TestPrecedence(t *testing.T) {
	tests := []struct {
		name     string
		op       string
		expected int
	}{
		// Valid precedences
		{"Power", "^", 5},
		{"Multiplication", "*", 4},
		{"Division", "/", 4},
		{"Addition", "+", 3},
		{"Subtraction", "-", 3},
		{"Greater than", ">", 2},
		{"Less than", "<", 2},
		{"Greater than or equal", ">=", 2},
		{"Less than or equal", "<=", 2},
		{"Equal", "==", 2},
		{"Logical AND", "&", 1},
		{"Logical OR", "|", 0},

		// Invalid operators
		{"Variable", "variable", -1},
		{"Number", "123", -1},
		{"Unary operator", "exp", -1},
		{"Min", "min", -1},
		{"Max", "max", -1},
		{"Parenthesis", "(", -1},
		{"Comma", ",", -1},
		{"Empty string", "", -1},
		{"Invalid operator", "++", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Precedence(tt.op)
			if result != tt.expected {
				t.Errorf("Precedence(%s) = %d, expected %d", tt.op, result, tt.expected)
			}
		})
	}
}

func TestIsLeftAssociative(t *testing.T) {
	tests := []struct {
		name     string
		op       string
		expected bool
	}{
		// Left associative operators
		{"Addition", "+", true},
		{"Subtraction", "-", true},
		{"Multiplication", "*", true},
		{"Division", "/", true},
		{"Greater than", ">", true},
		{"Less than", "<", true},
		{"Greater than or equal", ">=", true},
		{"Less than or equal", "<=", true},
		{"Equal", "==", true},
		{"Logical AND", "&", true},
		{"Logical OR", "|", true},

		// Right associative operators
		{"Power", "^", false},

		// Non-operators (should still return true by default)
		{"Variable", "variable", true},
		{"Number", "123", true},
		{"Unary operator", "exp", true},
		{"Min", "min", true},
		{"Max", "max", true},
		{"Parenthesis", "(", true},
		{"Comma", ",", true},
		{"Empty string", "", true},
		{"Invalid operator", "++", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsLeftAssociative(tt.op)
			if result != tt.expected {
				t.Errorf("IsLeftAssociative(%s) = %v, expected %v", tt.op, result, tt.expected)
			}
		})
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		name        string
		expression  string
		expected    []string
		expectError bool
	}{
		// Valid tokenization
		{"Simple expression", "x + y", []string{"x", "+", "y"}, false},
		{"With numbers", "x + 123", []string{"x", "+", "123"}, false},
		{"With decimals", "x + 12.34", []string{"x", "+", "12.34"}, false},
		{"With negative numbers", "x + -123", []string{"x", "+", "-123"}, false},
		{"With scientific notation", "x + 1.23e10", []string{"x", "+", "1.23e10"}, false},
		{"With parentheses", "(x + y)", []string{"(", "x", "+", "y", ")"}, false},
		{"With functions", "exp(x)", []string{"exp", "(", "x", ")"}, false},
		{"With commas", "min(x, y)", []string{"min", "(", "x", ",", "y", ")"}, false},
		{"Complex expression", "exp(x + y) * z", []string{"exp", "(", "x", "+", "y", ")", "*", "z"}, false},
		{"Comparison operators", "x >= y", []string{"x", ">=", "y"}, false},
		{"Equality operator", "x == y", []string{"x", "==", "y"}, false},
		{"Logical operators", "x & y", []string{"x", "&", "y"}, false},
		{"Logical OR", "x | y", []string{"x", "|", "y"}, false},
		{"Mixed operators", "x + y * z > w", []string{"x", "+", "y", "*", "z", ">", "w"}, false},
		{"Unary functions", "abs(x)", []string{"abs", "(", "x", ")"}, false},
		{"Nested functions", "exp(log(x))", []string{"exp", "(", "log", "(", "x", ")", ")"}, false},
		{"Multiple arguments", "min(x, y, z)", []string{"min", "(", "x", ",", "y", ",", "z", ")"}, false},
		{"No spaces", "x+y*z", []string{"x", "+", "y", "*", "z"}, false},
		{"Extra spaces", "  x   +   y  ", []string{"x", "+", "y"}, false},
		{"Variables with underscores", "var_name + other_var", []string{"var_name", "+", "other_var"}, false},
		{"Variables with numbers", "var123 + var456", []string{"var123", "+", "var456"}, false},
		{"Power operator", "x ^ y", []string{"x", "^", "y"}, false},
		{"All comparison operators", "x > y < z >= w <= u == v", []string{"x", ">", "y", "<", "z", ">=", "w", "<=", "u", "==", "v"}, false},

		// Edge cases
		{"Empty string", "", []string{}, false},
		{"Single number", "123", []string{"123"}, false},
		{"Single variable", "x", []string{"x"}, false},
		{"Single operator", "+", []string{"+"}, false},
		{"Single parenthesis", "(", []string{"("}, false},
		{"Just parentheses", "()", []string{"(", ")"}, false},
		{"Just comma", ",", []string{","}, false},
		{"Decimal number", "12.34", []string{"12.34"}, false},
		{"Negative decimal", "-12.34", []string{"-12.34"}, false},
		{"Scientific notation variants", "1.23e-10", []string{"1.23e-10"}, false},
		{"Scientific notation with E", "1.23E10", []string{"1.23E10"}, false},
		{"Scientific notation with plus", "1.23e+10", []string{"1.23e+10"}, false},

		// Invalid characters (should cause error)
		{"Invalid character hash", "x + y # z", []string{}, true},
		{"Invalid character at", "x @ y", []string{}, true},
		{"Invalid character dollar", "x $ y", []string{}, true},
		{"Invalid character percent", "x % y", []string{}, true},
		{"Invalid character tilde", "x ~ y", []string{}, true},
		{"Invalid character backtick", "x ` y", []string{}, true},
		{"Invalid character semicolon", "x ; y", []string{}, true},
		{"Invalid character colon", "x : y", []string{}, true},
		{"Invalid character question", "x ? y", []string{}, true},
		{"Invalid character square brackets", "x [y] z", []string{}, true},
		{"Invalid character curly braces", "x {y} z", []string{}, true},
		{"Invalid character backslash", "x \\ y", []string{}, true},
		{"Valid single pipe", "x | y", []string{"x", "|", "y"}, false},
		{"Valid single ampersand", "x & y", []string{"x", "&", "y"}, false},
		{"Invalid unicode", "x λ y", []string{}, true},
		{"Invalid quote", "x 'y' z", []string{}, true},
		{"Invalid double quote", "x \"y\" z", []string{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Tokenize(tt.expression)
			if tt.expectError {
				if err == nil {
					t.Errorf("Tokenize(%s) expected error but got none", tt.expression)
				}
			} else {
				if err != nil {
					t.Errorf("Tokenize(%s) unexpected error: %v", tt.expression, err)
				}
				if !equalSlices(result, tt.expected) {
					t.Errorf("Tokenize(%s) = %v, expected %v", tt.expression, result, tt.expected)
				}
			}
		})
	}
}

func TestInfixToPostfix(t *testing.T) {
	tests := []struct {
		name        string
		expression  string
		expected    []string
		expectError bool
	}{
		// Basic expressions
		{"Simple addition", "x + y", []string{"x", "y", "+"}, false},
		{"Simple subtraction", "x - y", []string{"x", "y", "-"}, false},
		{"Simple multiplication", "x * y", []string{"x", "y", "*"}, false},
		{"Simple division", "x / y", []string{"x", "y", "/"}, false},
		{"Simple power", "x ^ y", []string{"x", "y", "^"}, false},

		// Precedence tests
		{"Multiplication before addition", "x + y * z", []string{"x", "y", "z", "*", "+"}, false},
		{"Division before subtraction", "x - y / z", []string{"x", "y", "z", "/", "-"}, false},
		{"Power before multiplication", "x * y ^ z", []string{"x", "y", "z", "^", "*"}, false},
		{"Complex precedence", "x + y * z ^ w", []string{"x", "y", "z", "w", "^", "*", "+"}, false},

		// Associativity tests
		{"Left associative addition", "x + y + z", []string{"x", "y", "+", "z", "+"}, false},
		{"Left associative subtraction", "x - y - z", []string{"x", "y", "-", "z", "-"}, false},
		{"Left associative multiplication", "x * y * z", []string{"x", "y", "*", "z", "*"}, false},
		{"Left associative division", "x / y / z", []string{"x", "y", "/", "z", "/"}, false},
		{"Right associative power", "x ^ y ^ z", []string{"x", "y", "z", "^", "^"}, false},

		// Parentheses tests
		{"Simple parentheses", "(x + y)", []string{"x", "y", "+"}, false},
		{"Parentheses changing precedence", "(x + y) * z", []string{"x", "y", "+", "z", "*"}, false},
		{"Nested parentheses", "((x + y) * z)", []string{"x", "y", "+", "z", "*"}, false},
		{"Multiple parentheses", "(x + y) * (z - w)", []string{"x", "y", "+", "z", "w", "-", "*"}, false},
		{"Complex parentheses", "x + (y * (z + w))", []string{"x", "y", "z", "w", "+", "*", "+"}, false},

		// Function tests
		{"Unary function", "exp(x)", []string{"x", "exp"}, false},
		{"Unary function with expression", "exp(x + y)", []string{"x", "y", "+", "exp"}, false},
		{"Binary function", "min(x, y)", []string{"x", "y", "min"}, false},
		{"Binary function with expressions", "min(x + y, z * w)", []string{"x", "y", "+", "z", "w", "*", "min"}, false},
		{"Nested functions", "exp(log(x))", []string{"x", "log", "exp"}, false},
		{"Function in expression", "x + exp(y)", []string{"x", "y", "exp", "+"}, false},
		{"Multiple functions", "exp(x) + log(y)", []string{"x", "exp", "y", "log", "+"}, false},

		// Comparison operators
		{"Greater than", "x > y", []string{"x", "y", ">"}, false},
		{"Less than", "x < y", []string{"x", "y", "<"}, false},
		{"Greater than or equal", "x >= y", []string{"x", "y", ">="}, false},
		{"Less than or equal", "x <= y", []string{"x", "y", "<="}, false},
		{"Equal", "x == y", []string{"x", "y", "=="}, false},
		{"Comparison with arithmetic", "x + y > z", []string{"x", "y", "+", "z", ">"}, false},
		{"Complex comparison", "x + y > z * w", []string{"x", "y", "+", "z", "w", "*", ">"}, false},

		// Logical operators
		{"Logical AND", "x & y", []string{"x", "y", "&"}, false},
		{"Logical OR", "x | y", []string{"x", "y", "|"}, false},
		{"Logical with arithmetic", "x + y & z", []string{"x", "y", "+", "z", "&"}, false},
		{"Logical with comparison", "x > y & z < w", []string{"x", "y", ">", "z", "w", "<", "&"}, false},
		{"Complex logical", "x + y > z & w < u | v", []string{"x", "y", "+", "z", ">", "w", "u", "<", "&", "v", "|"}, false},

		// Numbers
		{"With integers", "x + 123", []string{"x", "123", "+"}, false},
		{"With decimals", "x + 12.34", []string{"x", "12.34", "+"}, false},
		{"With negative numbers", "x + -123", []string{"x", "-123", "+"}, false},
		{"With scientific notation", "x + 1.23e10", []string{"x", "1.23e10", "+"}, false},
		{"All numbers", "123 + 456", []string{"123", "456", "+"}, false},

		// Complex realistic expressions
		{"Scoring formula", "base_score + weight * factor", []string{"base_score", "weight", "factor", "*", "+"}, false},
		{"Normalization", "norm_min_max(score)", []string{"score", "norm_min_max"}, false},
		{"Conditional scoring", "score > threshold & active", []string{"score", "threshold", ">", "active", "&"}, false},
		{"Mathematical expression", "exp(log(x) + y) * z", []string{"x", "log", "y", "+", "exp", "z", "*"}, false},
		{"Min/max usage", "min(max(x, y), z)", []string{"x", "y", "max", "z", "min"}, false},

		// Edge cases
		{"Single number", "123", []string{"123"}, false},
		{"Single variable", "x", []string{"x"}, false},
		{"Empty parentheses", "()", []string{}, false},
		{"Function with no args", "exp()", []string{"exp"}, false},

		// Error cases
		{"Mismatched parentheses - missing close", "x + (y", []string{}, true},
		{"Mismatched parentheses - missing open", "x + y)", []string{}, true},
		{"Mismatched parentheses - extra close", "x + y))", []string{}, true},
		{"Mismatched parentheses - extra open", "((x + y)", []string{}, true},
		{"Misplaced comma", "x + , y", []string{}, true},
		{"Comma without function", "x, y", []string{}, true},
		{"Invalid token", "x + y # z", []string{}, true},
		{"Function with misplaced comma", "exp(,x)", []string{}, true},
		{"Function with trailing comma", "min(x,y,)", []string{}, true},
		{"Empty function call", "min(,)", []string{}, true},
		{"Nested mismatched parentheses", "exp((x + y)", []string{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := InfixToPostfix(tt.expression)
			if tt.expectError {
				if err == nil {
					t.Errorf("InfixToPostfix(%s) expected error but got none: result: %v", tt.expression, result)
				}
			} else {
				if err != nil {
					t.Errorf("InfixToPostfix(%s) unexpected error: %v", tt.expression, err)
				}
				if !equalSlices(result, tt.expected) {
					t.Errorf("InfixToPostfix(%s) = %v, expected %v", tt.expression, result, tt.expected)
				}
			}
		})
	}
}

func TestValidatePostfix(t *testing.T) {
	tests := []struct {
		name        string
		postfix     []string
		expectError bool
		errorMsg    string
	}{
		// Valid postfix expressions
		{"Simple addition", []string{"x", "y", "+"}, false, ""},
		{"Simple subtraction", []string{"x", "y", "-"}, false, ""},
		{"Simple multiplication", []string{"x", "y", "*"}, false, ""},
		{"Simple division", []string{"x", "y", "/"}, false, ""},
		{"Simple power", []string{"x", "y", "^"}, false, ""},
		{"Complex expression", []string{"x", "y", "z", "*", "+"}, false, ""},
		{"Unary function", []string{"x", "exp"}, false, ""},
		{"Binary function", []string{"x", "y", "min"}, false, ""},
		{"Nested operations", []string{"x", "y", "+", "z", "w", "*", "/"}, false, ""},
		{"Numbers", []string{"123", "456", "+"}, false, ""},
		{"Mixed numbers and variables", []string{"x", "123", "+"}, false, ""},
		{"Comparison", []string{"x", "y", ">"}, false, ""},
		{"Logical operation", []string{"x", "y", "&"}, false, ""},
		{"Complex logical", []string{"x", "y", ">", "z", "w", "<", "&"}, false, ""},
		{"Multiple unary functions", []string{"x", "exp", "y", "log", "+"}, false, ""},
		{"Function with binary operation", []string{"x", "y", "+", "exp"}, false, ""},
		{"Single value", []string{"x"}, false, ""},
		{"Single number", []string{"123"}, false, ""},

		// Invalid postfix expressions
		{"Not enough operands for binary", []string{"x", "+"}, true, "not enough operands for binary operator"},
		{"Not enough operands for function", []string{"exp"}, true, "not enough arguments for function"},
		{"Not enough operands for min", []string{"x", "min"}, true, "not enough arguments for function"},
		{"Too many operands", []string{"x", "y", "z"}, true, "leftover values"},
		{"Empty expression", []string{}, true, "empty stack"},
		{"Invalid token", []string{"x", "y", "invalid"}, true, "leftover values"},
		{"Multiple binary operators", []string{"x", "+", "+"}, true, "not enough operands for binary operator"},
		{"Function without enough args", []string{"x", "y", "z", "min"}, true, "leftover values"},
		{"Unary function with too many args", []string{"x", "y", "exp"}, true, "leftover values"},
		{"Mixed invalid", []string{"x", "y", "+", "z", "w"}, true, "leftover values"},
		{"Binary operator at start", []string{"+", "x", "y"}, true, "not enough operands for binary operator"},
		{"Function at start", []string{"exp", "x"}, true, "not enough arguments for function"},
		{"Multiple leftover values", []string{"x", "y", "z", "w", "+"}, true, "leftover values"},
		{"Consecutive operators", []string{"x", "y", "+", "*"}, true, "not enough operands for binary operator"},
		{"Function with no stack", []string{"x", "exp", "log"}, false, ""},
		{"Complex invalid", []string{"x", "y", "z", "+", "*", "w"}, true, "leftover values"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePostfix(tt.postfix)
			if tt.expectError {
				if err == nil {
					t.Errorf("ValidatePostfix(%v) expected error but got none", tt.postfix)
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("ValidatePostfix(%v) error message %q does not contain %q", tt.postfix, err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidatePostfix(%v) unexpected error: %v", tt.postfix, err)
				}
			}
		})
	}
}

func TestValidateExpression(t *testing.T) {
	tests := []struct {
		name        string
		expression  string
		expectError bool
		errorMsg    string
	}{
		// Valid expressions
		{"Simple addition", "x + y", false, ""},
		{"Simple subtraction", "x - y", false, ""},
		{"Simple multiplication", "x * y", false, ""},
		{"Simple division", "x / y", false, ""},
		{"Simple power", "x ^ y", false, ""},
		{"With parentheses", "(x + y) * z", false, ""},
		{"Unary function", "exp(x)", false, ""},
		{"Binary function", "min(x, y)", false, ""},
		{"Complex expression", "x + y * z - w / u", false, ""},
		{"Nested functions", "exp(log(x))", false, ""},
		{"Comparison", "x > y", false, ""},
		{"Logical operation", "x & y", false, ""},
		{"Complex logical", "x > y & z < w", false, ""},
		{"With numbers", "x + 123", false, ""},
		{"All numbers", "123 + 456", false, ""},
		{"Scientific notation", "x + 1.23e10", false, ""},
		{"Negative numbers", "x + -123", false, ""},
		{"Complex realistic", "exp(log(score) + weight) * factor", false, ""},
		{"Normalization", "norm_min_max(score)", false, ""},
		{"Conditional", "score > threshold & active", false, ""},
		{"Single value", "x", false, ""},
		{"Single number", "123", false, ""},

		// Invalid expressions - tokenization errors
		{"Invalid character", "x + y # z", true, "invalid characters"},
		{"Invalid character @", "x @ y", true, "invalid characters"},
		{"Invalid character %", "x % y", true, "invalid characters"},
		{"Invalid unicode", "x λ y", true, "invalid characters"},
		{"Invalid quotes", "x 'y' z", true, "invalid characters"},

		// Invalid expressions - parsing errors
		{"Mismatched parentheses", "x + (y", true, "mismatched parentheses"},
		{"Extra closing parenthesis", "x + y)", true, "mismatched parentheses"},
		{"Misplaced comma", "x + , y", true, "misplaced comma"},
		{"Comma without function", "x, y", true, "misplaced comma"},
		{"Invalid token sequence", "x + + y", true, ""},
		{"Empty parentheses with operator", "() + x", true, ""},
		{"Function with misplaced comma", "exp(,x)", true, "misplaced comma"},
		{"Function with trailing comma", "min(x,y,)", true, ""},

		// Invalid expressions - validation errors
		{"Incomplete expression", "x +", true, ""},
		{"Too many operands", "x y z", true, ""},
		{"Missing operand", "+ x", true, ""},
		{"Function without arguments", "exp() + x", true, ""},
		{"Binary function with one arg", "min(x)", true, ""},
		{"Unary function with multiple args", "exp(x, y)", true, ""},
		{"Operator without operands", "*", true, ""},
		{"Multiple operators", "x + * y", true, ""},
		{"Incomplete function call", "min(x,)", true, ""},
		{"Empty function call", "min()", true, ""},
		{"Function call with wrong arity", "min(x, y, z)", true, ""},

		// Edge cases
		{"Empty expression", "", true, "empty stack"},
		{"Only spaces", "   ", true, "empty stack"},
		{"Only operators", "+ - *", true, ""},
		{"Only parentheses", "()", true, ""},
		{"Only comma", ",", true, ""},
		{"Nested empty parentheses", "(())", true, ""},
		{"Multiple consecutive operators", "x + - * y", true, ""},
		{"Function name as variable", "min + x", true, ""},
		{"Operator as variable", "x + ++ y", true, ""},
		{"Mixed valid/invalid", "x + y $ z", true, "invalid characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExpression(tt.expression)
			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateExpression(%s) expected error but got none", tt.expression)
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("ValidateExpression(%s) error message %q does not contain %q", tt.expression, err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateExpression(%s) unexpected error: %v", tt.expression, err)
				}
			}
		})
	}
}

// Additional edge case tests for comprehensive coverage
func TestEdgeCases(t *testing.T) {
	// Test precedence edge cases
	t.Run("Precedence edge cases", func(t *testing.T) {
		tests := []struct {
			expression string
			expected   []string
		}{
			// Test all operator precedence levels
			{"x | y & z == w > u + v * w ^ z", []string{"x", "y", "z", "w", "==", "u", "v", "w", "z", "^", "*", "+", ">", "&", "|"}},
			// Test associativity edge cases
			{"x ^ y ^ z ^ w", []string{"x", "y", "z", "w", "^", "^", "^"}}, // Right associative
			{"x - y - z - w", []string{"x", "y", "-", "z", "-", "w", "-"}}, // Left associative
			{"x / y / z / w", []string{"x", "y", "/", "z", "/", "w", "/"}}, // Left associative
		}

		for _, tt := range tests {
			result, err := InfixToPostfix(tt.expression)
			if err != nil {
				t.Errorf("InfixToPostfix(%s) unexpected error: %v", tt.expression, err)
			}
			if !equalSlices(result, tt.expected) {
				t.Errorf("InfixToPostfix(%s) = %v, expected %v", tt.expression, result, tt.expected)
			}
		}
	})

	// Test function arity edge cases
	t.Run("Function arity edge cases", func(t *testing.T) {
		// Test that functions with different arities work correctly
		err := ValidateExpression("exp(x)")
		if err != nil {
			t.Errorf("ValidateExpression('exp(x)') unexpected error: %v", err)
		}

		err = ValidateExpression("min(x, y)")
		if err != nil {
			t.Errorf("ValidateExpression('min(x, y)') unexpected error: %v", err)
		}

		// Test incorrect arity
		err = ValidateExpression("exp(x, y)")
		if err == nil {
			t.Error("ValidateExpression('exp(x, y)') expected error but got none")
		}

		err = ValidateExpression("min(x)")
		if err == nil {
			t.Error("ValidateExpression('min(x)') expected error but got none")
		}
	})

	// Test deeply nested expressions
	t.Run("Deeply nested expressions", func(t *testing.T) {
		nested := "((((x + y) * z) - w) / u)"
		expected := []string{"x", "y", "+", "z", "*", "w", "-", "u", "/"}
		result, err := InfixToPostfix(nested)
		if err != nil {
			t.Errorf("InfixToPostfix(%s) unexpected error: %v", nested, err)
		}
		if !equalSlices(result, expected) {
			t.Errorf("InfixToPostfix(%s) = %v, expected %v", nested, result, expected)
		}
	})

	// Test complex function nesting
	t.Run("Complex function nesting", func(t *testing.T) {
		nested := "exp(log(abs(min(x, y))))"
		expected := []string{"x", "y", "min", "abs", "log", "exp"}
		result, err := InfixToPostfix(nested)
		if err != nil {
			t.Errorf("InfixToPostfix(%s) unexpected error: %v", nested, err)
		}
		if !equalSlices(result, expected) {
			t.Errorf("InfixToPostfix(%s) = %v, expected %v", nested, result, expected)
		}
	})
}

// Test comprehensive function arity validation
func TestComprehensiveFunctionArity(t *testing.T) {
	tests := []struct {
		name        string
		expression  string
		expectError bool
		description string
	}{
		// Test all unary functions with correct arity
		{"exp valid", "exp(x)", false, "exp should accept 1 argument"},
		{"log valid", "log(x)", false, "log should accept 1 argument"},
		{"abs valid", "abs(x)", false, "abs should accept 1 argument"},
		{"norm_min_max valid", "norm_min_max(x)", false, "norm_min_max should accept 1 argument"},
		{"percentile_rank valid", "percentile_rank(x)", false, "percentile_rank should accept 1 argument"},
		{"norm_percentile_0_99 valid", "norm_percentile_0_99(x)", false, "norm_percentile_0_99 should accept 1 argument"},
		{"norm_percentile_5_95 valid", "norm_percentile_5_95(x)", false, "norm_percentile_5_95 should accept 1 argument"},

		// Test binary functions with correct arity
		{"min valid", "min(x, y)", false, "min should accept 2 arguments"},
		{"max valid", "max(x, y)", false, "max should accept 2 arguments"},

		// Test unary functions with incorrect arity (too many)
		{"exp too many args", "exp(x, y)", true, "exp should not accept 2 arguments"},
		{"log too many args", "log(x, y)", true, "log should not accept 2 arguments"},
		{"abs too many args", "abs(x, y)", true, "abs should not accept 2 arguments"},
		{"norm_min_max too many args", "norm_min_max(x, y)", true, "norm_min_max should not accept 2 arguments"},
		{"percentile_rank too many args", "percentile_rank(x, y)", true, "percentile_rank should not accept 2 arguments"},
		{"norm_percentile_0_99 too many args", "norm_percentile_0_99(x, y)", true, "norm_percentile_0_99 should not accept 2 arguments"},
		{"norm_percentile_5_95 too many args", "norm_percentile_5_95(x, y)", true, "norm_percentile_5_95 should not accept 2 arguments"},

		// Test unary functions with no arguments
		{"exp no args", "exp()", true, "exp should not accept 0 arguments"},
		{"log no args", "log()", true, "log should not accept 0 arguments"},
		{"abs no args", "abs()", true, "abs should not accept 0 arguments"},
		{"norm_min_max no args", "norm_min_max()", true, "norm_min_max should not accept 0 arguments"},
		{"percentile_rank no args", "percentile_rank()", true, "percentile_rank should not accept 0 arguments"},
		{"norm_percentile_0_99 no args", "norm_percentile_0_99()", true, "norm_percentile_0_99 should not accept 0 arguments"},
		{"norm_percentile_5_95 no args", "norm_percentile_5_95()", true, "norm_percentile_5_95 should not accept 0 arguments"},

		// Test binary functions with incorrect arity
		{"min too few args", "min(x)", true, "min should not accept 1 argument"},
		{"max too few args", "max(x)", true, "max should not accept 1 argument"},
		{"min too many args", "min(x, y, z)", true, "min should not accept 3 arguments"},
		{"max too many args", "max(x, y, z)", true, "max should not accept 3 arguments"},
		{"min no args", "min()", true, "min should not accept 0 arguments"},
		{"max no args", "max()", true, "max should not accept 0 arguments"},

		// Test complex function combinations
		{"nested functions valid", "exp(log(abs(x)))", false, "nested unary functions should work"},
		{"mixed functions valid", "min(exp(x), log(y))", false, "mixed function types should work"},
		{"complex expression valid", "exp(x + y) + min(z, w)", false, "complex combinations should work"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExpression(tt.expression)
			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateExpression(%s) expected error but got none: %s", tt.expression, tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateExpression(%s) unexpected error: %v (%s)", tt.expression, err, tt.description)
				}
			}
		})
	}
}

// Test complex operator precedence and associativity combinations
func TestComplexOperatorInteractions(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		expected   []string
	}{
		// Test all precedence levels interacting
		{"All precedence levels", "a | b & c == d > e + f * g ^ h", []string{"a", "b", "c", "d", "==", "e", "f", "g", "h", "^", "*", "+", ">", "&", "|"}},

		// Test left associativity chains
		{"Long addition chain", "a + b + c + d + e", []string{"a", "b", "+", "c", "+", "d", "+", "e", "+"}},
		{"Long subtraction chain", "a - b - c - d - e", []string{"a", "b", "-", "c", "-", "d", "-", "e", "-"}},
		{"Long multiplication chain", "a * b * c * d * e", []string{"a", "b", "*", "c", "*", "d", "*", "e", "*"}},
		{"Long division chain", "a / b / c / d / e", []string{"a", "b", "/", "c", "/", "d", "/", "e", "/"}},

		// Test right associativity chains
		{"Long power chain", "a ^ b ^ c ^ d ^ e", []string{"a", "b", "c", "d", "e", "^", "^", "^", "^"}},

		// Test comparison operator chains
		{"Comparison chain", "a > b >= c < d <= e == f", []string{"a", "b", ">", "c", ">=", "d", "<", "e", "<=", "f", "=="}},

		// Test logical operator chains
		{"Logical AND chain", "a & b & c & d", []string{"a", "b", "&", "c", "&", "d", "&"}},
		{"Logical OR chain", "a | b | c | d", []string{"a", "b", "|", "c", "|", "d", "|"}},
		{"Mixed logical", "a | b & c | d & e", []string{"a", "b", "c", "&", "|", "d", "e", "&", "|"}},

		// Test precedence with parentheses
		{"Parentheses override precedence", "(a + b) * (c - d)", []string{"a", "b", "+", "c", "d", "-", "*"}},
		{"Nested parentheses precedence", "((a + b) * c) ^ (d - e)", []string{"a", "b", "+", "c", "*", "d", "e", "-", "^"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := InfixToPostfix(tt.expression)
			if err != nil {
				t.Errorf("InfixToPostfix(%s) unexpected error: %v", tt.expression, err)
			}
			if !equalSlices(result, tt.expected) {
				t.Errorf("InfixToPostfix(%s) = %v, expected %v", tt.expression, result, tt.expected)
			}
		})
	}
}

// Test boundary conditions and stress tests
func TestBoundaryConditions(t *testing.T) {
	t.Run("Very long expressions", func(t *testing.T) {
		// Test with a very long but valid expression
		expr := "x"
		expected := []string{"x"}
		for i := 0; i < 100; i++ {
			expr += " + y" + fmt.Sprintf("%d", i)
			expected = append(expected, "y"+fmt.Sprintf("%d", i), "+")
		}

		result, err := InfixToPostfix(expr)
		if err != nil {
			t.Errorf("InfixToPostfix(long expression) unexpected error: %v", err)
		}
		if !equalSlices(result, expected) {
			t.Errorf("InfixToPostfix(long expression) failed")
		}
	})

	t.Run("Deeply nested parentheses", func(t *testing.T) {
		// Test with deeply nested parentheses
		expr := "x"
		expected := []string{"x"}
		for i := 0; i < 50; i++ {
			expr = "(" + expr + ")"
		}

		result, err := InfixToPostfix(expr)
		if err != nil {
			t.Errorf("InfixToPostfix(deeply nested) unexpected error: %v", err)
		}
		if !equalSlices(result, expected) {
			t.Errorf("InfixToPostfix(deeply nested) = %v, expected %v", result, expected)
		}
	})

	t.Run("Many function calls", func(t *testing.T) {
		// Test with many nested function calls
		expr := "x"
		expected := []string{"x"}
		functions := []string{"exp", "log", "abs", "norm_min_max", "percentile_rank"}

		for i := 0; i < 20; i++ {
			fn := functions[i%len(functions)]
			expr = fn + "(" + expr + ")"
			expected = append(expected, fn)
		}

		result, err := InfixToPostfix(expr)
		if err != nil {
			t.Errorf("InfixToPostfix(many functions) unexpected error: %v", err)
		}
		if !equalSlices(result, expected) {
			t.Errorf("InfixToPostfix(many functions) = %v, expected %v", result, expected)
		}
	})
}
