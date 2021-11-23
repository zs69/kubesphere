package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	licensetypes "kubesphere.io/kubesphere/pkg/simple/client/license/types/v1alpha1"
	"kubesphere.io/kubesphere/pkg/utils/idutils"
)

var outputFile string

// environment can be dev or prod
var environment string
var duration time.Duration
var startDate string
var endDate string
var maEnd string
var keyFile string
var licenseType string

var ls = licensetypes.License{}

func main() {
	cmd := newCmd(os.Args[1:])
	if err := cmd.Execute(); err != nil {
		klog.Errorf("error: %s", err)
	}
}

func newCmd(args []string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen-license",
		Short: "gen-license generate a valid license",
		RunE: func(cmd *cobra.Command, args []string) error {
			if keyFile == "" {
				if environment == "prod" {
					keyFile = "./cert-tools/cert/ks-apiserver-key.pem"
				} else {
					keyFile = "./cert-tools/cert/ks-apiserver-dev-key.pem"
				}
			}
			key, err := ioutil.ReadFile(keyFile)
			if err != nil {
				klog.Errorf("read key file failed, error: %s", err)
				return err
			}

			if ls.LicenseId == "" {
				ls.LicenseId = idutils.GetUuid36("")
			}
			ls.LicenseType = licensetypes.LicenseType(licenseType)

			now := time.Now().UTC()
			ls.IssueAt = now

			if ls.LicenseType == licensetypes.LicenseTypeMaintenance {
				ls.MaxCluster = 0
				if maEnd == "" {
					return fmt.Errorf("the ma-end should not be empty")
				}
				t, err := time.ParseInLocation("2006-01-02", maEnd, time.UTC)
				if err != nil {
					klog.Errorf("parse start time failed, error: %s", err)
					return err
				}
				ls.MaintenanceEnd = &t
			}

			if startDate != "" && endDate != "" {
				t, err := time.ParseInLocation("2006-01-02", startDate, time.UTC)
				if err != nil {
					klog.Errorf("parse start time failed, error: %s", err)
					return err
				}
				ls.NotBefore = &t

				t, err = time.ParseInLocation("2006-01-02", endDate, time.UTC)
				if err != nil {
					klog.Errorf("parse end time failed, error: %s", err)
					return err
				}
				end := t.Add(24*time.Hour - time.Second)
				ls.NotAfter = &end
			} else if startDate != "" && duration != 0 {
				t, err := time.ParseInLocation("2006-01-02", startDate, time.UTC)
				if err != nil {
					klog.Errorf("parse start time failed, error: %s", err)
					return err
				}
				ls.NotBefore = &t
				end := t.Add(duration)
				ls.NotAfter = &end
			} else {
				ls.NotBefore = &now
				end := now.Add(duration)
				ls.NotAfter = &end
			}

			// license for ma is valid forever.
			if ls.LicenseType == licensetypes.LicenseTypeMaintenance {
				ls.NotAfter = nil
			}

			ls.Version = 1
			err = ls.Sign(key)
			if err != nil {
				klog.Fatalf("sign license failed, error: %s", err)
			}
			data, err := json.Marshal(ls)
			if err != nil {
				klog.Fatalf("json marshal failed, error: %s", err)
			}

			if outputFile != "" {
				err = ioutil.WriteFile(outputFile, data, os.FileMode(0644))
				if err != nil {
					return err
				}
			} else {
				fmt.Fprintln(os.Stdout, data)
			}

			err = licenseRecord(data)
			if err != nil {
				klog.Error("log license record failed")
				return err
			}

			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&keyFile, "key", "ks-apiserver-key.pem", "the key to sign the license")
	f.StringVar(&environment, "env", "prod", "environment to build, prod or dev")
	f.StringVar(&outputFile, "output", "license.out", "output file")
	f.StringVar(&licenseType, "type", "", "type of the license, valid value: subscription, managed, maintenance")
	f.IntVar(&ls.MaxNode, "max-node", 0, "max node")
	f.IntVar(&ls.MaxCluster, "max-cluster", 1, "max cluster")
	f.IntVar(&ls.MaxCPU, "max-cpu", 0, "max cpu num")
	f.IntVar(&ls.MaxCore, "max-core", 0, "max core")
	f.StringVar(&ls.Subject.Name, "user.name", "", "license's user name")
	f.StringVar(&ls.Subject.Corporation, "user.co", "", "license's user corporation")
	f.StringVar(&ls.Issuer.Name, "issuer.name", "qingcloud", "license's issuer name")
	f.StringVar(&ls.Issuer.Corporation, "issuer.co", "qingcloud", "license's issuer corporation")
	f.StringVar(&ls.LicenseId, "license-id", "", "id of the license, if empty, will generate a random id")
	f.StringVar(&ls.StartVersion, "start-version", "", "the start version this license can be applied to")
	f.StringVar(&ls.EndVersion, "end-version", "", "the end version this license can not be applied to")

	f.StringVar(&startDate, "start-date", "", "the start date this license can work, format: 2021-07-22")
	f.StringVar(&endDate, "end-date", "", "the end date this license can work")
	f.StringVar(&maEnd, "ma-end", "", "the end date maintenance")

	f.DurationVarP(&duration, "duration", "d", 365*24*time.Hour, "valid duration, default value is 1 year")
	return cmd
}

func licenseRecord(data []byte) error {
	now := time.Now()
	day := now.Format("2006-01-02")
	fileName := fmt.Sprintf("%s_%s_%s", "license", environment, day)

	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	f.Write(data)
	f.WriteString("\n")
	return nil
}
