package v1alpha1

import (
	"encoding/json"
	"testing"

	"k8s.io/client-go/util/cert"
	"k8s.io/klog/v2"
)

var licenseData = `{"l_id":"95mw5mxryqnmmy","l_type":"ma","ver":1,"subj":{"co":"","name":"lihui"},"issuer":{"co":"qingcloud","name":"qingcloud"},"not_before":"2021-07-22T00:00:00Z","not_after":"2021-11-16T23:59:59Z","ma_end":"0001-01-01T00:00:00Z","issue_at":"2021-11-17T02:17:25.756174Z","max_cluster":2,"max_node":2,"max_cpu":134,"max_core":9,"sig":"SvPolcDc0X8U2fji30dWH7ZMFpjXNotH/Ujs/GAyTyAhO0OiPtPYljazFOXNLYEycNpokjp7HifpBVRtdFxgoe8Eq8YwsL4S1MoqO7+Ur2mgLcg+dJcoxcLL1Yzo0vo8o1ibalBkLJ3j9ruoam9zqQThj1mIMi9qYXMbES9ri+SGgg67cWt706u/bVKTY2kzrm2uACU6v6LzyVPnJTVM0Q0wSiKlc7mbSgmdZB7YDhyvA33DO0D0oQrUQ6CTyPcLFhqqbSvWoN4drySpyDw8biF6nVO10JwSl6vPdod4RHmMMuSjjREiATrmGWTmQaKZ5QHRADAIDp65VbcBNjK5jhOb6O4MJLXKnTGWdgiisiH48jKZS2qLJcLYvyZD5Fij61q8UsLHniQaGn32kjitl2NoLCXXLFL3mXCtxuNfja0UiurqZ7KrXu3O3Ou3ehonZuuqbkRqDLWmKlnmVK1ZE27wFNAvxa5HB+4RtbN8Pg/vytENIDLJFa1YOxKj5S7ZGYLaXYVOyiCysUwMe4HWxkphIbFUH4HLopokVv01M6jkycUtjmKGDwuve5zkjYjzBAhTC5s8/kYz77vNKpJCORyOD4+J8lgce2NN7I6ipIs7P3883bSWCitwdefwTdgJXD7rlVa6Q2f5lHcMl1vBhf9RVWxbEz8YtZdnT6Pg7pQ="}`

func TestVerify(t *testing.T) {
	certData := []byte(`-----BEGIN CERTIFICATE-----
MIIGqjCCBJKgAwIBAgIUQPx9c3cWxp1KtMG20pg0+1AY+O0wDQYJKoZIhvcNAQEN
BQAwYzELMAkGA1UEBhMCQ04xCzAJBgNVBAgTAkhCMQswCQYDVQQHEwJXSDESMBAG
A1UEChMJUWluZ0Nsb3VkMRMwEQYDVQQLEwpLdWJlc3BoZXJlMREwDwYDVQQDEwhr
cy1jbG91ZDAeFw0yMTA2MTcwMjI3MDBaFw0yMjA2MTcwMjI3MDBaMGcxCzAJBgNV
BAYTAkNOMQswCQYDVQQIEwJIQjELMAkGA1UEBxMCV0gxEjAQBgNVBAoTCVFpbmdD
bG91ZDETMBEGA1UECxMKS3ViZXNwaGVyZTEVMBMGA1UEAxMMa3MtYXBpc2VydmVy
MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAuloAn8bSXgOmP1Kacwfc
dgm1u33htIu4444qk6I6fJgmamybuEAaIxIEQVbqmtr/4Zjd0Pxwb9z2SEghSIg+
Y8xkjaEmIUa90p9PR9mLBveHRkvC5XiJaE8cXu5CcBhq364v73lZXP3qsvEdgIW6
CFK/J+tJ03TpmB+729tEjXZeQXzCqoeRoJZfCqVt4bkwN1pkn38oM1usNG1xX+ab
xb3ppqjSCKWfNTzF6SuLfx/xQj5T6hzRW6aSbLKTGOafrZQvJLf9CfcxEVAE/vFV
ZyvfK6zKNv5QhpclRgw7aDXhJm8cuNv9IVLgvPCeyJrv5waMKlXe5tM6RiLSp22v
7axCrmFhLjG/Qa3zUWniPw5JtxwPpBNJJ0vxsMFXo2lTppxlppbt2WdI+0Gr/evP
O5Wl86dpVbNv3d4lw9NTxEb0azB2P2W006oyBUmqOaVPqcb30X5r6E79QKsnZzZq
Dy1FRTT8XVN0F0LUVpqOaeqfPu8HfrJzFyFCChrj8pkgXvcVCLqwSWU3XNOgwUQJ
J8TZucvqdcKNbCAJi7SaDXkee1XudYgXKU9bLDICAM5orabrl4oUh/nhNVW5KHge
QeJSwv4s/El41bnNdWuwDOitdF3QXfVqbti13m6ZJlX8GPNNouHPrp2yprBe1qko
AJxEQsoq4gUykJnivn7RwGcCAwEAAaOCAVAwggFMMA4GA1UdDwEB/wQEAwIFoDAd
BgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwDAYDVR0TAQH/BAIwADAdBgNV
HQ4EFgQUe2oNo+nO2GFYggq6d8xiK9VLHXMwHwYDVR0jBBgwFoAUb0J+gs4sfCU7
dC+vjN4unqfXnZswgcwGA1UdEQSBxDCBwYIJbG9jYWxob3N0ggxrcy1hcGlzZXJ2
ZXKCHmtzLWFwaXNlcnZlci5rdWJlc3BoZXJlLXN5c3RlbYIia3MtYXBpc2VydmVy
Lmt1YmVzcGhlcmUtc3lzdGVtLnN2Y4Iqa3MtYXBpc2VydmVyLmt1YmVzcGhlcmUt
c3lzdGVtLnN2Yy5jbHVzdGVygjBrcy1hcGlzZXJ2ZXIua3ViZXNwaGVyZS1zeXN0
ZW0uc3ZjLmNsdXN0ZXIubG9jYWyHBH8AAAEwDQYJKoZIhvcNAQENBQADggIBALVY
SJKPDFX706QxWZqKJ++hvs/HwgH501egU8v1Ye+ug58o+Xz6fzOqhdxSm1kQ5Qs7
2B647K+IxBEbP5FTLp/CsKtt/4FJo4kr6346ktn2vPR+xrEMTrE4mFMR9tA3EDL5
r62UNWzrZk0SpIdZWhXqlXM2voJ1Sqz+tYINURfxwOQi7dtmK5uXMfe62NJBJ8MB
YFOK+cSNfzvgMYAW0AGz8tSjYht7DqfszbCCSMhn7fEUxgkBIHUd/TMWoHD6ZKLL
acWUXlf+1afwsqjg+JJLmyH+laYgW5BTFnjNEU4UJ6z6Nus9Q3kzPM5enU4QdFXM
fmrSDZDfvmhDzd9eiBAqNFF6CCbwg3fN8ic7bJKxHqbPfxrCSW7C7TW1culERc/S
SN/MtrLnMSgoIffic3/+1ICsEXJOalCKlG/li4sk65to3KWDy7w5tTOexprq0nwB
8lOCdzY1Kaw2HQVZVAjioZoo0ovHCR9LPvGuAmLRzJRomgSguVs9EpHhFz4F+a5B
CzT4sWWPuUY7TaZEmpZIjRHCvlI4T2Djwy1WlKRIcbSAAW3M7mJMW2YjFJ1kMIw+
Yphz8wDbOpsLHe4kJJogEZzgrvMNiMEWTfx22pXSyM9NcY9FSnrHwZw55MJDJ4fB
CKvyZg+HPQd1KtRWMYVHlLfH2NsWeEh/oDolq00E
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
