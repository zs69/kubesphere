package v1alpha1

import (
	"testing"

	"k8s.io/klog"
)

func TestDeprecatedLicense(t *testing.T) {
	var lid = "11l40j0mko8l43"
	if _, exists := DeprecatedLicense[lid]; !exists {
		klog.Errorf("deprecated license not found")
		t.Fail()
	}
}
