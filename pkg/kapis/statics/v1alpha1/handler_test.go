package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_randStaticsFileName(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		wantStyle   string
	}{
		{"png", "images/png", ".png"},
		{"jpg", "images/jpeg", ".jpg"},
		{"svg", "images/svg+xml", ".svg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFilename, gotStyle := randStaticsFileName(tt.contentType)
			assert.NotNil(t, gotFilename)
			if gotStyle != tt.wantStyle {
				t.Errorf("randStaticsFileName() gotStyle = %v, want %v", gotStyle, tt.wantStyle)
			}
		})
	}
}
