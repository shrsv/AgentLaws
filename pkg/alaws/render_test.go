package alaws

import (
	"regexp"
	"testing"
)

func TestExportFileName(t *testing.T) {
	tests := []struct {
		title string
		want  string
	}{
		{"Engineering Governance", "engineering-governance"},
		{"  Payments & Refunds!! ", "payments-refunds"},
		{"Combined Lawbook", "combined-lawbook"},
		{"", "lawbook"},
		{"###", "lawbook"},
	}
	dateRe := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	for _, tt := range tests {
		got := ExportFileName(tt.title, "pdf")
		wantPrefix := tt.want + "-"
		if len(got) <= len(wantPrefix) || got[:len(wantPrefix)] != wantPrefix || got[len(got)-4:] != ".pdf" {
			t.Errorf("ExportFileName(%q, \"pdf\") = %q, want prefix %q and suffix .pdf", tt.title, got, wantPrefix)
			continue
		}
		date := got[len(wantPrefix) : len(got)-4]
		if !dateRe.MatchString(date) {
			t.Errorf("ExportFileName(%q, \"pdf\") = %q, date part %q doesn't look like YYYY-MM-DD", tt.title, got, date)
		}
	}
}
