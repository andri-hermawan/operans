package domain

import "testing"

func TestValidKey(t *testing.T) {
	for k, want := range map[string]bool{
		"dashboard":                  true,
		"report/report-ritase-daily": true,
		"upload/weighbridge":         true,
		"":                           false,
		"/dashboard":                 false,
		"Dashboard":                  false,
		"report ritase":              false,
		"a;drop table":               false,
	} {
		if got := ValidKey(k); got != want {
			t.Errorf("ValidKey(%q) = %v, mau %v", k, got, want)
		}
	}
}

func TestEffective_UserOverridesRole(t *testing.T) {
	role := []Rule{{PageKey: "report/ritase", Allowed: false}, {PageKey: "tracking", Allowed: true}}
	user := []Rule{{PageKey: "report/ritase", Allowed: true}}
	got := Effective(role, user)
	if !got["report/ritase"] || !got["tracking"] || len(got) != 2 {
		t.Fatalf("aturan user harus menimpa role: %v", got)
	}
}
