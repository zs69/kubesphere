package license

import compbasemetrics "k8s.io/component-base/metrics"

var (
	LicenseValidityPeriod = compbasemetrics.NewGaugeVec(
		&compbasemetrics.GaugeOpts{
			Name:           "kubesphere_enterprise_license_validity_seconds",
			Help:           "KubeSphere Enterprise license validity period in seconds. A value less than zero means the license has already expired.",
			StabilityLevel: compbasemetrics.ALPHA,
		},
		[]string{"type"},
	)
)
