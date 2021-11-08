/*
Copyright 2020 The KubeSphere Authors.

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
	"io"
	"time"
)

type Client interface {
	SearchNotifications(filter *Filter, sort, order string, from, size int64) (*Notifications, error)
	ExportNotifications(filter *Filter, sort, order string, w io.Writer) error
}

type Filter struct {
	AlertName      []string
	AlertNameFuzzy []string
	AlertType      []string
	AlertTypeFuzzy []string
	Severity       []string
	SeverityFuzzy  []string
	Namespace      []string
	NamespaceFuzzy []string
	Service        []string
	ServiceFuzzy   []string
	Container      []string
	ContainerFuzzy []string
	Pod            []string
	PodFuzzy       []string
	MessageFuzzy   []string
	StartTime      time.Time
	EndTime        time.Time
}

type Notifications struct {
	Total int64         `json:"total" description:"total number of matched results"`
	Items []interface{} `json:"items" description:"actual array of results"`
}
