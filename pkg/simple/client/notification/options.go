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

package notification

import (
	"github.com/spf13/pflag"

	"kubesphere.io/kubesphere/pkg/utils/reflectutils"
)

type History struct {
	Enable      bool   `json:"enable" yaml:"enable"`
	Host        string `json:"host" yaml:"host"`
	BasicAuth   bool   `json:"basicAuth" yaml:"basicAuth"`
	Username    string `json:"username" yaml:"username"`
	Password    string `json:"password" yaml:"password"`
	IndexPrefix string `json:"indexPrefix,omitempty" yaml:"indexPrefix"`
	Version     string `json:"version" yaml:"version"`
}

type Options struct {
	Endpoint string
	History  `json:"history,omitempty" yaml:"history,omitempty"`
}

func NewNotificationOptions() *Options {
	return &Options{
		History: History{
			Host:        "",
			IndexPrefix: "ks-logstash-notification",
			Version:     "",
		},
	}
}

func (s *Options) ApplyTo(options *Options) {
	if options == nil {
		options = s
		return
	}

	if s.History.Host != "" {
		reflectutils.Override(options, s)
	}
}

func (s *Options) Validate() []error {
	errs := make([]error, 0)
	return errs
}

func (s *Options) AddFlags(fs *pflag.FlagSet, c *Options) {
	fs.BoolVar(&s.Enable, "notification-history-enabled", c.Enable, "Enable notification history component or not. ")

	fs.BoolVar(&s.BasicAuth, "notification-history-elasticsearch-basicAuth", c.BasicAuth, ""+
		"Does elasticsearch basic auth enabled.")

	fs.StringVar(&s.Username, "notification-history-elasticsearch-username", c.Username, ""+
		"ElasticSearch authentication username")

	fs.StringVar(&s.Password, "notification-history-elasticsearch-password", c.Password, ""+
		"ElasticSearch authentication passwor")

	fs.StringVar(&s.Host, "notification-history-elasticsearch-host", c.Host, ""+
		"Elasticsearch service host.")

	fs.StringVar(&s.IndexPrefix, "notification-history-index-prefix", c.IndexPrefix, ""+
		"Index name prefix. KubeSphere will retrieve notification history against indices matching the prefix.")

	fs.StringVar(&s.Version, "notification-history-elasticsearch-version", c.Version, ""+
		"Elasticsearch major version, e.g. 5/6/7, if left blank, will detect automatically."+
		"Currently, minimum supported version is 5.x")
}
