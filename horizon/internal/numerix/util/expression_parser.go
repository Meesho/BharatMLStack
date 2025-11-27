package util

import (
	"strconv"
	"strings"
)

var binaryOps = map[string]bool{
	"+": true, "-": true, "*": true, "/": true, "^": true,
	">": true, "<": true, ">=": true, "<=": true, "==": true,
	"min": true, "max": true, "&": true, "|": true,
}

var unaryOps = map[string]bool{
	"exp": true, "log": true, "abs": true,
	"norm_min_max": true, "percentile_rank": true,
	"norm_percentile_0_99": true, "norm_percentile_5_95": true,
}

func IsOp(token string) bool {
	return binaryOps[token] || unaryOps[token]
}

func IsNumber(token string) bool {
	_, err := strconv.ParseFloat(token, 64)
	return err == nil
}

func ExtractVariables(expression string) []string {
	tokens := strings.Fields(expression)
	vars := make([]string, 0)

	for _, token := range tokens {
		if IsNumber(token) || IsOp(token) {
			continue
		}
		// Preserve all occurrences, including duplicates and parentheses
		vars = append(vars, token)
	}

	return vars

}
