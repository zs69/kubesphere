package v1alpha1

import "testing"

func TestPlatformUIConf_Valid(t *testing.T) {
	type fields struct {
		Title       string
		Description string
		Logo        string
		Favicon     string
		Background  string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{"1", fields{"KubeSphere", "xxxx", "", "", ""}, false},
		{"2", fields{"_KubeSphere", "xxxx", "", "", ""}, false},
		{"3", fields{"kubesphere1_", "xxxx", "", "", ""}, false},
		{"4", fields{"kube-sphere", "xxxx", "", "", ""}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PlatformUIConf{
				Title:       tt.fields.Title,
				Description: tt.fields.Description,
				Logo:        tt.fields.Logo,
				Favicon:     tt.fields.Favicon,
				Background:  tt.fields.Background,
			}
			if got := p.Valid(); got != tt.want {
				t.Errorf("Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}
