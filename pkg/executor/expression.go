package executor

import (
	"fmt"
	"strings"
)

// EvaluateRouteExpression evaluates a simple string expression against a value.
// Supported expressions:
//
//	== "value"      - exact match
//	!= "value"      - not equal
//	contains "value" - substring match
//	startsWith "value" - prefix match
//	endsWith "value" - suffix match
//
// Operands must be quoted with double quotes.
// Returns error for unrecognized operators, missing quotes, or empty expressions.
func EvaluateRouteExpression(value string, expression string) (bool, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return false, fmt.Errorf("empty expression")
	}

	// Find the first quote — everything before it is the operator,
	// content between first and last quote is the operand.
	firstQuote := strings.Index(expression, "\"")
	if firstQuote == -1 {
		return false, fmt.Errorf("expression operand must be quoted: %s", expression)
	}

	operator := strings.TrimSpace(expression[:firstQuote])
	rest := expression[firstQuote+1:]

	lastQuote := strings.LastIndex(rest, "\"")
	if lastQuote == -1 {
		return false, fmt.Errorf("expression operand must be quoted: %s", expression)
	}

	operand := rest[:lastQuote]

	switch operator {
	case "==":
		return value == operand, nil
	case "!=":
		return value != operand, nil
	case "contains":
		return strings.Contains(value, operand), nil
	case "startsWith":
		return strings.HasPrefix(value, operand), nil
	case "endsWith":
		return strings.HasSuffix(value, operand), nil
	default:
		return false, fmt.Errorf("unknown operator: %s", operator)
	}
}
