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
	"testing"
	"time"

	"kubesphere.io/kubesphere/pkg/simple/client/notification"

	"github.com/stretchr/testify/assert"
)

func TestParseToQueryPart(t *testing.T) {
	q := `
{
  "query":{
    "bool":{
      "filter":[
        {
          "bool":{
            "should":[
              {
                "match_phrase":{
                  "alertname.keyword":"CPUThrottlingHigh"
                }
              },
              {
                "match_phrase":{
                  "alertname.keyword":"KubePodCrashLooping"
                }
              }
            ],
            "minimum_should_match":1
          }
        },
        {
          "bool":{
            "should":[
              {
                "wildcard":{
                  "alertname.keyword":"*Kube*"
                }
              }
            ],
            "minimum_should_match":1
          }
        },
        {
          "bool":{
            "should":[
              {
                "match_phrase":{
                  "alerttype.keyword":"auditing"
                }
              },
              {
                "match_phrase":{
                  "alerttype.keyword":"events"
                }
              }
            ],
            "minimum_should_match":1
          }
        },
        {
          "bool":{
            "should":[
              {
                "wildcard":{
                  "alerttype.keyword":"*a*"
                }
              }
            ],
            "minimum_should_match":1
          }
        },
        {
          "bool":{
            "should":[
              {
                "match_phrase":{
                  "severity.keyword":"error"
                }
              },
              {
                "match_phrase":{
                  "severity.keyword":"critical"
                }
              }
            ],
            "minimum_should_match":1
          }
        },
        {
          "bool":{
            "should":[
              {
                "wildcard":{
                  "severity.keyword":"*b*"
                }
              }
            ],
            "minimum_should_match":1
          }
        },
        {
          "bool":{
            "should":[
              {
                "match_phrase":{
                  "namespace.keyword":"kubesphere-system"
                }
              },
              {
                "match_phrase":{
                  "namespace.keyword":"kube-system"
                }
              }
            ],
            "minimum_should_match":1
          }
        },
        {
          "bool":{
            "should":[
              {
                "wildcard":{
                  "namespace.keyword":"*system*"
                }
              }
            ],
            "minimum_should_match":1
          }
        },
        {
          "bool":{
            "should":[
              {
                "match_phrase":{
                  "service.keyword":"ks-apiserver"
                }
              },
              {
                "match_phrase":{
                  "service.keyword":"ks-console"
                }
              }
            ],
            "minimum_should_match":1
          }
        },
        {
          "bool":{
            "should":[
              {
                "wildcard":{
                  "service.keyword":"*ks*"
                }
              }
            ],
            "minimum_should_match":1
          }
        },
        {
          "bool":{
            "should":[
              {
                "match_phrase":{
                  "pod.keyword":"ks-apiserver"
                }
              },
              {
                "match_phrase":{
                  "pod.keyword":"kube-apiserver"
                }
              }
            ],
            "minimum_should_match":1
          }
        },
        {
          "bool":{
            "should":[
              {
                "wildcard":{
                  "pod.keyword":"*apiserver*"
                }
              }
            ],
            "minimum_should_match":1
          }
        },
        {
          "bool":{
            "should":[
              {
                "match_phrase":{
                  "container.keyword":"ks-apiserver"
                }
              },
              {
                "match_phrase":{
                  "container.keyword":"kube-apiserver"
                }
              }
            ],
            "minimum_should_match":1
          }
        },
        {
          "bool":{
            "should":[
              {
                "wildcard":{
                  "container.keyword":"*apiserver*"
                }
              }
            ],
            "minimum_should_match":1
          }
        },
        {
          "bool":{
            "should":[
              {
                "wildcard":{
                  "message.keyword":"*throttling of CPU in namespace*"
                }
              }
            ],
            "minimum_should_match":1
          }
        },
        {
          "range":{
            "notificationTime":{
              "gte":"2019-12-01T01:01:01.000000001Z",
              "lte":"2020-01-01T01:01:01.000000001Z"
            }
          }
        }
      ]
    }
  }
}
`
	nsCreateTime := time.Date(2020, time.Month(1), 1, 1, 1, 1, 1, time.UTC)
	startTime := nsCreateTime.AddDate(0, -1, 0)
	endTime := nsCreateTime.AddDate(0, 0, 0)

	filter := &notification.Filter{
		AlertName:      []string{"CPUThrottlingHigh", "KubePodCrashLooping"},
		AlertNameFuzzy: []string{"Kube"},
		AlertType:      []string{"auditing", "events"},
		AlertTypeFuzzy: []string{"a"},
		Severity:       []string{"error", "critical"},
		SeverityFuzzy:  []string{"b"},
		Namespace:      []string{"kubesphere-system", "kube-system"},
		NamespaceFuzzy: []string{"system"},
		Service:        []string{"ks-apiserver", "ks-console"},
		ServiceFuzzy:   []string{"ks"},
		Container:      []string{"ks-apiserver", "kube-apiserver"},
		ContainerFuzzy: []string{"apiserver"},
		Pod:            []string{"ks-apiserver", "kube-apiserver"},
		PodFuzzy:       []string{"apiserver"},
		MessageFuzzy:   []string{"throttling of CPU in namespace"},
		StartTime:      startTime,
		EndTime:        endTime,
	}

	qp := ParseToQueryPart(filter)
	bs, err := json.Marshal(qp)
	if err != nil {
		panic(err)
	}

	queryPart := &map[string]interface{}{}
	if err := json.Unmarshal(bs, queryPart); err != nil {
		panic(err)
	}
	expectedQueryPart := &map[string]interface{}{}
	if err := json.Unmarshal([]byte(q), expectedQueryPart); err != nil {
		panic(err)
	}

	assert.Equal(t, expectedQueryPart, queryPart)
}
