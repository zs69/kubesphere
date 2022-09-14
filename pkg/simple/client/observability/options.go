/*
Copyright 2020 KubeSphere Authors

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

package observability

import (
	"github.com/spf13/pflag"

	"kubesphere.io/kubesphere/pkg/utils/reflectutils"
)

type Options struct {
	Monitoring *MonitoringOptions `json:"monitoring,omitempty" yaml:"monitoring" mapstructure:"monitoring"`
}

type MonitoringOptions struct {
	Endpoint string `json:"endpoint,omitempty" yaml:"endpoint"`
}

func NewObservabilityOptions() *Options {
	return &Options{
		Monitoring: &MonitoringOptions{
			Endpoint: "",
		},
	}
}

func (s *Options) Validate() []error {
	var errs []error
	return errs
}

func (s *Options) ApplyTo(options *Options) {
	if s.Monitoring != nil {
		reflectutils.Override(options, s)
	}
}

func (s *Options) AddFlags(fs *pflag.FlagSet, c *Options) {
	fs.StringVar(&s.Monitoring.Endpoint, "whizard-endpoint", c.Monitoring.Endpoint, ""+
		"Whizard monitoring service endpoint which stores observability monitoring data for multiple clusters.")
}
