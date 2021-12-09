# If you have the rsa key-pair, you can run it to generate a license.
# If you do not have the key-pair, you should generate it and replace the pkg/license/ks-apiserver.pem with the
# the public key you generated, then generate the license file.

DEP_DIR=/usr/bin

KEY_FILE ?= ks-apiserver

generate-qkcp-key: generate-ca
	# generate qkcp key file
	if [[ ! -f cert/${KEY_FILE}-key.pem ]]; then ${DEP_DIR}/cfssl gencert -config cert-config/ca-config.json -profile ks -ca cert/ca.pem -ca-key cert/ca-key.pem cert-config/ks-apiserver-csr.json |  ${DEP_DIR}/cfssljson -bare cert/${KEY_FILE}; fi

generate-ca: 
	if [[ ! -d ./cert ]]; then mkdir cert; fi	
	# generate ca file
	if [[ ! -f cert/ca-key.pem ]]; then ${DEP_DIR}/cfssl genkey -config cert-config/ca-config.json -initca ./cert-config/ca-csr.json | ${DEP_DIR}/cfssljson -bare cert/ca; fi

MAX_CORE ?= 100
MAX_CLUSTER ?= 2
MAX_NODE ?= 20
USER_NAME ?= testing
generate-license: generate-qkcp-key
	# generate license for test
	go run gen-license/main.go  --user.name ${USER_NAME} --key ./cert/ks-apiserver-key.pem --start-date $(shell date +%F) --max-node ${MAX_NODE} --max-cluster ${MAX_CLUSTER} --max-core ${MAX_CORE} --type subscription

OS ?= $(shell go env GOOS)
ARCH ?= $(shell go env GOARCH)

download-dep:
	curl -L -o ${DEP_DIR}/cfssl https://github.com/cloudflare/cfssl/releases/download/v1.6.1/cfssl-bundle_1.6.1_${OS}_${ARCH} && chmod +x ${DEP_DIR}/cfssl 
	curl -L -o ${DEP_DIR}/cfssl-json https://github.com/cloudflare/cfssl/releases/download/v1.6.1/cfssljson_1.6.1_${OS}_${ARCH} && chmod +x  ${DEP_DIR}/cfssljson 
