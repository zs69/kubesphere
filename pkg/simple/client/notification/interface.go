/*
Copyright 2021 The KubeSphere Authors.

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
	// The name of the alert that trigger the notification.
	AlertName      []string
	AlertNameFuzzy []string
	// The type of the alert that trigger the notification, "auditing", "events" and so on.
	AlertType      []string
	AlertTypeFuzzy []string
	// The severity of the alert that trigger the notification, "normal", "warning" and "ciritial".
	Severity      []string
	SeverityFuzzy []string
	// The namespace of the alert that trigger the notification.
	Namespace      []string
	NamespaceFuzzy []string
	// The service that triggered the alert.
	Service      []string
	ServiceFuzzy []string
	// The container that triggered the alert.
	Container      []string
	ContainerFuzzy []string
	// The pod that triggered the alert.
	Pod          []string
	PodFuzzy     []string
	MessageFuzzy []string
	StartTime    time.Time
	EndTime      time.Time
}

type Notifications struct {
	Total int64         `json:"total" description:"total number of matched results"`
	Items []interface{} `json:"items" description:"actual array of results"`
}
