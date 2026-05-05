package util

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

var binaryOpsPrecedence = map[string]int{
	"^": 5,
	"*": 4, "/": 4,
	"+": 3, "-": 3,
	">": 2, "<": 2, ">=": 2, "<=": 2, "==": 2,
	"&": 1,
	"|": 0,
}

var funcArity = map[string]int{
	"exp": 1,
	"log": 1,
	"abs": 1,

	"min": 2,
	"max": 2,

	"norm_min_max":         1,
	"percentile_rank":      1,
	"norm_percentile_0_99": 1,
	"norm_percentile_5_95": 1,
}

func IsUnaryOperator(token string) bool {
	_, ok := funcArity[token]
	return ok
}

func IsOperator(token string) bool {
	_, isBinary := binaryOps[token]
	isFunc := IsUnaryOperator(token)
	return isBinary || isFunc
}

func isVariable(token string) bool {
	return !IsNumber(token) &&
		token != "(" && token != ")" && token != "," &&
		!IsOperator(token)
}

func IsBinaryOperator(token string) bool {
	_, ok := binaryOpsPrecedence[token]
	return ok
}

func Precedence(op string) int {
	if prec, ok := binaryOpsPrecedence[op]; ok {
		return prec
	}
	return -1
}

func IsLeftAssociative(op string) bool {
	return op != "^"
}

func Tokenize(expr string) ([]string, error) {
	// Updated regex to properly handle scientific notation
	re := regexp.MustCompile(`\s*(-?\d+\.?\d*[eE][+-]?\d+|-?\d*\.\d+|-?\d+|[A-Za-z_][A-Za-z0-9_]*|>=|<=|==|\||&|[+\-*/^<>()=,])\s*`)
	tokens := re.FindAllStringSubmatch(expr, -1)

	var result []string
	for _, match := range tokens {
		token := strings.TrimSpace(match[1])
		if token != "" {
			result = append(result, token)
		}
	}

	joined := strings.Join(result, "")
	cleaned := strings.ReplaceAll(expr, " ", "")
	if joined != cleaned {
		return nil, errors.New("invalid characters in expression")
	}

	return result, nil
}

func InfixToPostfix(expression string) ([]string, error) {
	tokens, err := Tokenize(expression)
	if err != nil {
		return nil, err
	}

	var output []string
	var stack []string

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]

		switch {
		case IsNumber(token) || isVariable(token):
			output = append(output, token)

		case IsUnaryOperator(token):
			stack = append(stack, token)

		case token == ",":
			// Check for leading comma (comma right after opening parenthesis)
			if i > 0 && tokens[i-1] == "(" {
				return nil, errors.New("misplaced comma: comma cannot appear immediately after opening parenthesis")
			}

			// Check for trailing comma (comma right before closing parenthesis)
			if i < len(tokens)-1 && tokens[i+1] == ")" {
				return nil, errors.New("misplaced comma: comma cannot appear immediately before closing parenthesis")
			}

			// Check for consecutive commas
			if i > 0 && tokens[i-1] == "," {
				return nil, errors.New("misplaced comma: consecutive commas are not allowed")
			}

			for len(stack) > 0 && stack[len(stack)-1] != "(" {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			if len(stack) == 0 {
				return nil, errors.New("misplaced comma or mismatched parentheses")
			}

		case IsBinaryOperator(token):
			for len(stack) > 0 {
				top := stack[len(stack)-1]
				if IsOperator(top) && ((IsLeftAssociative(token) && Precedence(token) <= Precedence(top)) ||
					(!IsLeftAssociative(token) && Precedence(token) < Precedence(top))) {
					output = append(output, top)
					stack = stack[:len(stack)-1]
				} else {
					break
				}
			}
			stack = append(stack, token)

		case token == "(":
			stack = append(stack, token)

		case token == ")":
			for len(stack) > 0 && stack[len(stack)-1] != "(" {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			if len(stack) == 0 {
				return nil, errors.New("mismatched parentheses")
			}
			stack = stack[:len(stack)-1]

			if len(stack) > 0 && IsUnaryOperator(stack[len(stack)-1]) {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}

		default:
			return nil, errors.New("invalid token: " + token)
		}
	}

	for len(stack) > 0 {
		top := stack[len(stack)-1]
		if top == "(" || top == ")" {
			return nil, errors.New("mismatched parentheses")
		}
		output = append(output, top)
		stack = stack[:len(stack)-1]
	}

	return output, nil
}

func ValidatePostfix(postfix []string) error {
	var stack []string

	for i, token := range postfix {
		if minArgs, isFunc := funcArity[token]; isFunc {
			if len(stack) < minArgs {
				errMsg := "validation error: not enough arguments for function '" + token + "'. "
				errMsg += "Expected at least " + strconv.Itoa(minArgs) + " argument(s), but stack only had " + strconv.Itoa(len(stack)) + " value(s) available"
				contextStart := i - minArgs - 2
				if contextStart < 0 {
					contextStart = 0
				}
				contextEnd := i + 1
				if contextEnd > len(postfix) {
					contextEnd = len(postfix)
				}
				errMsg += " (context: " + strings.Join(postfix[contextStart:contextEnd], " ") + ")"

				return errors.New(errMsg)
			}
			popCount := minArgs
			stack = stack[:len(stack)-popCount]

			stack = append(stack, "value")

		} else {
			switch {
			case IsNumber(token) || isVariable(token):
				stack = append(stack, "value")

			case IsBinaryOperator(token):
				if len(stack) < 2 {
					return errors.New("validation error: not enough operands for binary operator '" + token + "'")
				}
				stack = stack[:len(stack)-2]
				stack = append(stack, "value")

			default:
				return errors.New("validation error: unknown or misplaced token in postfix expression '" + token + "'")
			}
		}
	}

	if len(stack) == 1 {
		return nil
	} else if len(stack) == 0 {
		return errors.New("validation error: invalid postfix expression, resulted in empty stack (maybe empty input?)")
	} else {
		return errors.New("validation error: invalid postfix expression, stack has leftover values (" + strconv.Itoa(len(stack)) + " items), structure is incorrect")
	}
}

func ValidateExpression(expression string) error {
	postfix, err := InfixToPostfix(expression)
	if err != nil {
		return err
	}
	return ValidatePostfix(postfix)
}
