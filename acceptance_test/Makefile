UNAME := $(shell uname)
CLUSTER_NAME := kubelogin-acceptance-test

KUBECONFIG := testdata/kubeconfig.yaml:kubeconfig.yaml
export KUBECONFIG

all:
	$(MAKE) clean
	$(MAKE) create-cluster
	$(MAKE) deploy
	$(MAKE) import-ca-certificate
	$(MAKE) test

.PHONY: clean
clean:
	-rm -fr testdata/

# Kubernetes cluster
.PHONY: create-cluster delete-cluster
create-cluster:
	mkdir -p testdata/
	kind create cluster --name $(CLUSTER_NAME) --kubeconfig testdata/kubeconfig.yaml --config cluster.yaml

delete-cluster:
	kind delete cluster --name $(CLUSTER_NAME)

# Authentication and authorization
.PHONY: deploy
deploy: testdata/server.crt testdata/server.key
	kubectl apply -f dex.yaml
	kubectl -n dex create secret tls dex.tls --cert=testdata/server.crt --key=testdata/server.key
	kubectl -n dex rollout status deployment server
	kubectl apply -f role.yaml

# TLS certificate for dex server
testdata/ca.key:
	docker cp $(CLUSTER_NAME)-control-plane:/etc/kubernetes/pki/ca.key testdata/ca.key

testdata/ca.crt:
	docker cp $(CLUSTER_NAME)-control-plane:/etc/kubernetes/pki/ca.crt testdata/ca.crt

testdata/server.key:
	openssl genrsa -out $@ 2048

testdata/server.csr: openssl.cnf testdata/server.key
	openssl req -new -key testdata/server.key -out $@ -subj "/CN=server.dex.svc.cluster.local" -config openssl.cnf
	openssl req -noout -text -in $@

# expiration must not be longer than 825 days on macOS.
# https://superuser.com/questions/1492207/neterr-cert-revoked-in-chrome-chromium-introduced-with-macos-catalina

testdata/server.crt: openssl.cnf testdata/server.csr testdata/ca.crt testdata/ca.key
	openssl x509 -req -in testdata/server.csr -CA testdata/ca.crt -CAkey testdata/ca.key -CAcreateserial -out $@ -sha256 -days 730 -extensions v3_req -extfile openssl.cnf
	openssl x509 -text -in $@

.PHONY: import-certificate
import-ca-certificate: testdata/ca.crt
ifeq ($(UNAME), Darwin)
	@echo "NOTE: you need to open the keychain and add the certificate: $(CURDIR)/testdata/ca.crt"
	@echo "      press enter to proceed..."
	read
else
	sudo mkdir -v /usr/local/share/ca-certificates/kubernetes
	sudo cp -v testdata/ca.crt /usr/local/share/ca-certificates/kubernetes/
	sudo update-ca-certificates
	mkdir -p $(HOME)/.pki/nssdb
	certutil -d sql:$(HOME)/.pki/nssdb -A -n kubernetes -i testdata/ca.crt -t TCP,TCP,TCP
endif

.PHONY: test
test:
	go test -v .
