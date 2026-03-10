package executor

import (
	"testing"
)

func TestEvaluateRouteExpression(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		expression string
		want       bool
		wantErr    string
	}{
		// == operator
		{
			name:       "equal match",
			value:      "billing",
			expression: `== "billing"`,
			want:       true,
		},
		{
			name:       "equal no match",
			value:      "shipping",
			expression: `== "billing"`,
			want:       false,
		},

		// != operator
		{
			name:       "not equal match",
			value:      "shipping",
			expression: `!= "billing"`,
			want:       true,
		},
		{
			name:       "not equal no match",
			value:      "billing",
			expression: `!= "billing"`,
			want:       false,
		},

		// contains operator
		{
			name:       "contains match",
			value:      "high priority issue",
			expression: `contains "high priority"`,
			want:       true,
		},
		{
			name:       "contains no match",
			value:      "low priority issue",
			expression: `contains "high priority"`,
			want:       false,
		},

		// startsWith operator
		{
			name:       "startsWith match",
			value:      "billing department",
			expression: `startsWith "billing"`,
			want:       true,
		},
		{
			name:       "startsWith no match",
			value:      "the billing department",
			expression: `startsWith "billing"`,
			want:       false,
		},

		// endsWith operator
		{
			name:       "endsWith match",
			value:      "support ticket",
			expression: `endsWith "ticket"`,
			want:       true,
		},
		{
			name:       "endsWith no match",
			value:      "ticket support",
			expression: `endsWith "ticket"`,
			want:       false,
		},

		// Case sensitivity
		{
			name:       "case sensitive exact match fails",
			value:      "Billing",
			expression: `== "billing"`,
			want:       false,
		},
		{
			name:       "case sensitive contains fails",
			value:      "HIGH PRIORITY",
			expression: `contains "high"`,
			want:       false,
		},

		// Empty value with empty operand
		{
			name:       "empty value equals empty operand",
			value:      "",
			expression: `== ""`,
			want:       true,
		},
		{
			name:       "non-empty value not equal to empty operand",
			value:      "something",
			expression: `== ""`,
			want:       false,
		},

		// Values with spaces
		{
			name:       "value with spaces contains match",
			value:      "this is a high priority task",
			expression: `contains "high priority"`,
			want:       true,
		},

		// Error: empty expression
		{
			name:       "error on empty expression",
			value:      "test",
			expression: "",
			wantErr:    "empty expression",
		},
		{
			name:       "error on whitespace-only expression",
			value:      "test",
			expression: "   ",
			wantErr:    "empty expression",
		},

		// Error: missing quotes
		{
			name:       "error on missing quotes",
			value:      "test",
			expression: "== billing",
			wantErr:    "expression operand must be quoted: == billing",
		},

		// Error: unknown operator
		{
			name:       "error on unknown operator",
			value:      "test",
			expression: `matches "foo"`,
			wantErr:    "unknown operator: matches",
		},
		{
			name:       "error on empty operator",
			value:      "test",
			expression: `"foo"`,
			wantErr:    "unknown operator: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateRouteExpression(tt.value, tt.expression)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("EvaluateRouteExpression(%q, %q) = %v, want %v", tt.value, tt.expression, got, tt.want)
			}
		})
	}
}
