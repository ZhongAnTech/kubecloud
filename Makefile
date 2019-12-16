.PHONY: all build

INSTALL_DIR?=./kubecloud
GO_TAGS?=
CGO_ENABLED?=0
GO_PACKAGES?=./...

GO_BUILD=go build -v -tags="${GO_TAGS}"

export CGO_ENABLED := ${CGO_ENABLED}

all: build

build:
	${GO_BUILD} ./cmd/kubecloud

lint:
	go vet -structtag=false -tags="${GO_TAGS}" ${GO_PACKAGES}

# NOTE: cgo is required by go test
test:
	CGO_ENABLED=1 go test -race -cover -failfast -vet=off -tags="${GO_TAGS}" ${GO_PACKAGES}

install: build
	mkdir -p ${INSTALL_DIR}
	mkdir -p ${INSTALL_DIR}/log
	cp -Rp kubecloud conf apidoc ${INSTALL_DIR}

docker:
	docker build -t kubecloud .

clean:
	rm -rf cli web
	go clean -cache