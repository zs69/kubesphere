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

package elasticsearch

import (
	"bytes"
	"fmt"
	"io"

	"kubesphere.io/kubesphere/pkg/simple/client/notification"
	"kubesphere.io/kubesphere/pkg/utils/stringutils"

	jsoniter "github.com/json-iterator/go"

	"kubesphere.io/kubesphere/pkg/simple/client/es"
	"kubesphere.io/kubesphere/pkg/simple/client/es/query"
)

const (
	DefaultSortField = "notificationTime"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type client struct {
	c *es.Client
}

func (c *client) SearchNotifications(filter *notification.Filter, sort, order string, from, size int64) (*notification.Notifications, error) {

	b := query.NewBuilder().
		WithQuery(ParseToQueryPart(filter)).
		WithSort(formatSort(sort), order).
		WithFrom(from).
		WithSize(size)

	resp, err := c.c.Search(b, filter.StartTime, filter.EndTime, false)
	if err != nil || resp == nil {
		return nil, err
	}

	notifications := &notification.Notifications{Total: c.c.GetTotalHitCount(resp.Total)}
	for _, hit := range resp.AllHits {
		notifications.Items = append(notifications.Items, hit.Source)
	}
	return notifications, nil
}

func (c *client) ExportNotifications(filter *notification.Filter, sort, order string, w io.Writer) error {

	var id string
	var data []interface{}

	b := query.NewBuilder().
		WithQuery(ParseToQueryPart(filter)).
		WithSort(formatSort(sort), order).
		WithFrom(0).
		WithSize(1000)

	resp, err := c.c.Search(b, filter.StartTime, filter.EndTime, true)
	if err != nil {
		return err
	}

	defer c.c.ClearScroll(id)

	id = resp.ScrollId

	for _, hit := range resp.AllHits {
		data = append(data, hit.Source)
	}

	// limit to retrieve max 100k records
	for i := 0; i < 100; i++ {
		if i != 0 {
			data, id, err = c.scroll(id)
			if err != nil {
				return err
			}
		}
		if len(data) == 0 {
			return nil
		}

		output := new(bytes.Buffer)
		for _, d := range data {
			bs, err := jsoniter.Marshal(d)
			if err != nil {
				return err
			}
			output.WriteString(stringutils.StripAnsi(string(bs)))
		}
		_, err = io.Copy(w, output)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *client) scroll(id string) ([]interface{}, string, error) {
	resp, err := c.c.Scroll(id)
	if err != nil {
		return nil, id, err
	}

	var data []interface{}
	for _, hit := range resp.AllHits {
		data = append(data, hit.Source)
	}
	return data, resp.ScrollId, nil
}

func NewClient(options *notification.Options) (notification.Client, error) {
	c := &client{}

	var err error
	c.c, err = es.NewClient(options.History.Host, options.BasicAuth, options.Username, options.Password, options.IndexPrefix, options.Version)
	return c, err
}

func formatSort(sort string) string {

	if sort == "" || sort == DefaultSortField ||
		sort == "startsAs" || sort == "endsAs" {
		return DefaultSortField
	}

	return fmt.Sprintf("%s.keyword", sort)
}

func ParseToQueryPart(f *notification.Filter) *query.Query {
	if f == nil {
		return nil
	}

	var mini int32 = 1
	b := query.NewBool()

	appendParameter := func(key string, val, fuzzyVal []string) {
		b.AppendFilter(query.NewBool().
			AppendMultiShould(query.NewMultiMatchPhrase(key, val)).
			WithMinimumShouldMatch(mini))

		bi := query.NewBool().WithMinimumShouldMatch(mini)
		for _, v := range fuzzyVal {
			bi.AppendShould(query.NewWildcard(key, fmt.Sprintf("*"+v+"*")))
		}
		b.AppendFilter(bi)
	}

	appendParameter("alertname.keyword", f.AlertName, f.AlertNameFuzzy)
	appendParameter("alerttype.keyword", f.AlertType, f.AlertTypeFuzzy)
	appendParameter("severity.keyword", f.Severity, f.SeverityFuzzy)
	appendParameter("namespace.keyword", f.Namespace, f.NamespaceFuzzy)
	appendParameter("service.keyword", f.Service, f.ServiceFuzzy)
	appendParameter("pod.keyword", f.Pod, f.PodFuzzy)
	appendParameter("container.keyword", f.Container, f.ContainerFuzzy)
	appendParameter("message.keyword", nil, f.MessageFuzzy)

	r := query.NewRange("notificationTime")
	if !f.StartTime.IsZero() {
		r.WithGTE(f.StartTime)
	}
	if !f.EndTime.IsZero() {
		r.WithLTE(f.EndTime)
	}

	b.AppendFilter(r)

	return query.NewQuery().WithBool(b)
}
