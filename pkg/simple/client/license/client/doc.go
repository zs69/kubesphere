// Package client instantiate a client to connect to ks-apisever and fetch the license.
// If user run `go build` with `-tags "client_check_license"`, then at startup, the binary will check license. If the
// license is empty or not valid, the binary will exit.
package client
