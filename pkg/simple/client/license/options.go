/*
Copyright 2021 KubeSphere Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package license

import (
	"github.com/spf13/pflag"
)

type Options struct {
	SkipLicenseCheck bool `json:"skip-license-check,omitempty" yaml:"skip-license-check,omitempty"`
}

func NewOptions() *Options {
	return &Options{
		SkipLicenseCheck: false,
	}
}

// AddFlags add options flags to command line flags,
func (s *Options) AddFlags(fs *pflag.FlagSet, c *Options) {
	// We will skip the license check only when the user sets this field to true.
	fs.BoolVar(&s.SkipLicenseCheck, "skip-license-check", c.SkipLicenseCheck, "Skip license check")
	fs.MarkHidden("skip-license-check")
}
