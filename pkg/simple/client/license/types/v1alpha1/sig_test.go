package v1alpha1

import (
	"encoding/json"
	"testing"

	"k8s.io/client-go/util/cert"
	"k8s.io/klog/v2"
)

var licenseData = `{"licenseId":"44n6mnv6wqm17n","licenseType":"subscription","version":1,"subject":{"co":"","name":"lihui"},"issuer":{"co":"qingcloud","name":"qingcloud"},"notBefore":"2021-07-22T00:00:00Z","notAfter":"2044-05-15T08:00:00Z","issueAt":"2021-11-23T11:55:32.793746Z","maxCluster":1,"maxNode":3,"maxCore":5,"signature":"ICsYi8TkRW6JZ4a5T8Fu62aDo7HZ4ekknz1yavYAOpX5r1cqI9IZGY1asHgUvPb1LfwMLT/Ej3ermR7bOLOhhAPYMiLZubu39BLCIFYWwNK8cHlxi999jyfxMlTA4zyDVsMEu3c6PDGXwu+CZQaCyoqzturHUcrazAxh7v2EX3rhkVvQuILU9fQzIDG6lLwmUZLQ3G0ckwQ90C1ImAMKFSBi1AUXOA6VT9eCPtShJCGfqSP4hqAYH1Zb+ZmWKakNfffKgw7Tlmeymclu2xeVTADmRYoGsGttIAGaXFLXOx9e8q8X26vnnYXGLvWurkXPE8itvVUotM2LomfvBca2lUDyleHiQSkcFdlJEQ55ithE67w8bdg+MuIejbsYpzIjFhewCuUCeqQKIxE2YVQlUic07K8YudVgfMJA09vCBaZcbUENrRK4KTI0zjAHWAx6OjU9d1EUTAPfxD9gRQswMEg1XUQmASqUIFR20i+rC2U3pdFRHxZk2Xh2pFTXXldzDplh1T7ftBGDQyz5mou0kX8zuxbIcC/kYT7QLh80+A42EzILzG7jcR5hTrUiWizKjsyP5TTqxUwjPo9bXMmyURsoD5wMiQ2hIPWTPwOjygCt/6LsR5kzqVQQznqEn3RQOVqTZ4UZPOMcWW8GLqytPHe555IS8N0KrH32laZyIx4="}`

func TestVerify(t *testing.T) {
	certData := []byte(`-----BEGIN CERTIFICATE-----
MIIGqjCCBJKgAwIBAgIUW44xd5QPQrQnTYUvdmSKGM/KahIwDQYJKoZIhvcNAQEN
BQAwYzELMAkGA1UEBhMCQ04xCzAJBgNVBAgTAkhCMQswCQYDVQQHEwJXSDESMBAG
A1UEChMJUWluZ0Nsb3VkMRMwEQYDVQQLEwpLdWJlc3BoZXJlMREwDwYDVQQDEwhr
cy1jbG91ZDAeFw0yMTA3MTkwOTU3MDBaFw0yMjA3MTkwOTU3MDBaMGcxCzAJBgNV
BAYTAkNOMQswCQYDVQQIEwJIQjELMAkGA1UEBxMCV0gxEjAQBgNVBAoTCVFpbmdD
bG91ZDETMBEGA1UECxMKS3ViZXNwaGVyZTEVMBMGA1UEAxMMa3MtYXBpc2VydmVy
MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAxR24Gf0yEaxvT9EV6VWX
zXMNZy8H5SngO9SRy4wqYGY6F3adoDjHFFvF5fCdItKApIJ70wt0a/DrK1uZtdFU
CVIAg5sDPXSWwSyqYyeOoVl/h7b6JUUgKEyBt1lrdoGNsBOt/3LoSij3XRc9ApfI
TSYInblPio5JnJPw/hRoU9+5rjPaujoU3ILFrBjZ6hE633UonLzgf6GtvPglKpye
GWTHu5KB2BH2/LOJQiiiNITO4haEm1p3zYZQcVRm2VkBYhzTfBCw7zfUahDrsyQy
cYRvWVnTBaMP15yNhBVcI/RyJD+3VcIUC9N7gv6YEd76tD7LOO6E1WRXxDrn78kE
qsa+6FSTnn2O9W63tzdyn9URB7aEfy1NqscLZ40Z6In39CMRuIcIgyqi2g5aSsLr
ZpFoZpnn9BlqFyZocervMQrEEIHh2AKEO19sThAMcU6wWr6gxsqqakQ8x8z0qQnr
UPiTDDvcfFzlcy0iKIDmaUldb/oTvEGBg/sAYlo5Zh9qa1ct9mInUGcgary9wPXi
Ou1boky4T3s3KbAiXx0ZFESVmhNM6vi/5F0P1QGBln2TR5TV3otbH2RSprDiSzPD
2tX5sJYr3lk3hXhdWLsqJs1JecePlrJBiRoXvpxpkSPX+Q6+Aj+b8IIVD1XN93nA
/GX9QnEpdxeWIUpz9A04lwECAwEAAaOCAVAwggFMMA4GA1UdDwEB/wQEAwIFoDAd
BgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwDAYDVR0TAQH/BAIwADAdBgNV
HQ4EFgQU2pVZFqLEIr0V3CltbYVTJ0+ezjcwHwYDVR0jBBgwFoAUb0J+gs4sfCU7
dC+vjN4unqfXnZswgcwGA1UdEQSBxDCBwYIJbG9jYWxob3N0ggxrcy1hcGlzZXJ2
ZXKCHmtzLWFwaXNlcnZlci5rdWJlc3BoZXJlLXN5c3RlbYIia3MtYXBpc2VydmVy
Lmt1YmVzcGhlcmUtc3lzdGVtLnN2Y4Iqa3MtYXBpc2VydmVyLmt1YmVzcGhlcmUt
c3lzdGVtLnN2Yy5jbHVzdGVygjBrcy1hcGlzZXJ2ZXIua3ViZXNwaGVyZS1zeXN0
ZW0uc3ZjLmNsdXN0ZXIubG9jYWyHBH8AAAEwDQYJKoZIhvcNAQENBQADggIBAL7S
c+J+v10Em9otLJ/vipbfdvqLp/EJIYV4HCy0ocQa8lS0urutsIx2mV320KwIXg/R
jRbRuBlI2rcoi33WQoQ8e/ia7awWY9SXi2dnMIy1J4Nh853fMb+M4jbeD/ruBWwG
I4/j6rirMl5Jy5ZW6Sh4I17akJTG3jS0mZmDpHNeYUXBLF16dUEptyCtY8tFBHc3
fNOG/ZeBNQuNkQGVNKIOczdVpfSiMamEV9EJZBZ3y8QuKTa/GKhm/Tv1SBMi1ukg
clHxno2cgGx6D93efh0uwkucG92B2XPQxAoL8mCqGv4cQXa1BtOZJOjy2qBBJI6Y
KJloEyAClcTM42s8Df8FXihuSzXMcl5/zPduH+RrA6LtjM8DANkcuXKNXeTdEhKv
MWfcSKYwh9N/5W2YYX+Qys0yrgtRoHsqL8aINKJKeZfgRCuenac3gzV8QS4aBtkO
pLaAB4KDAz2wjz0lzHcbqh9vS03Z2xQSkUPPEwtxW6L3v0b4UFxmBWk8A877eXB7
aZIn39AnHAblNHgTiWdHkNLXoBrJfBvQvQE4W3OIVY5bDD5etvYXg2Ia7g9YPaCU
wFpsK8TpSHZ3WrEmoyWBtS1qlyFKobW6ryPE3jHaxuyEzGl/fjoBv1m28JG1T05P
emRTtBKKc2xA/ldubc3CAR4qJ/pR1LvfH8I6NN8l
-----END CERTIFICATE-----`)

	var license License
	err := json.Unmarshal([]byte(licenseData), &license)
	if err != nil {
		klog.Errorf("json unmarshal, error: %s", err)
		t.Fail()
	}
	certs, err := cert.ParseCertsPEM(certData)
	if err != nil {
		klog.Errorf("parse cert failed, error:%s", err)
		t.Fail()
	}
	ok, vio := license.Verify(certs[0])
	if vio != nil {
		klog.Errorf("verify failed, error: %s", vio.Reason)
		t.Fail()
	}

	if ok {
		klog.Infof("verify success")
	} else {
		klog.Errorf("verify failed")
		t.Fail()
	}
}
