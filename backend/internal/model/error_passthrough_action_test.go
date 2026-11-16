package model

import "testing"

func TestErrorPassthroughEffectiveActionLegacyCompatibility(t *testing.T) {
	for _, tc := range []struct {
		code, body bool
		want string
	}{
		{true, true, ErrorActionPassthrough},
		{false, true, ErrorActionRewrite},
		{true, false, ErrorActionRewrite},
		{false, false, ErrorActionRewrite},
	} {
		rule := &ErrorPassthroughRule{PassthroughCode: tc.code, PassthroughBody: tc.body}
		if got := rule.EffectiveAction(); got != tc.want { t.Fatalf("flags %v/%v: got %q, want %q", tc.code, tc.body, got, tc.want) }
	}
}

func TestErrorPassthroughActionValidation(t *testing.T) {
	for _, action := range []string{ErrorActionPassthrough, ErrorActionRewrite, ErrorActionRetry, "loop"} {
		t.Run(action, func(t *testing.T) {
			rule := &ErrorPassthroughRule{Name:"rule", Action:action, MatchMode:MatchModeAny, ErrorCodes:[]int{499}}
			err := rule.Validate()
			wantError := action == ErrorActionRewrite || action == "loop"
			if (err != nil) != wantError { t.Fatalf("Validate()=%v, want error=%v", err, wantError) }
		})
	}
}
