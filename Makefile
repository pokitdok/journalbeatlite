SHELL := /bin/bash

HASH := $(shell git rev-parse --short head)
DATE := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LIB_HASH := $(shell cd ../libbeatlite ; git rev-parse --short head)

.PHONY: test
test:
	go test -bench=. -v .

.PHONY: update
update:
	go get -u github.com/coreos/go-systemd/sdjournal

.PHONY: install
install:
	-rm /home/ubuntu/go/bin/journalbeatlite
	go install -ldflags '-X main.BuildHash=$(HASH) -X main.BuildDate=$(DATE) -X main.LibBuildHash=$(LIB_HASH)' .

.PHONY: upload
upload: install
	aws s3 cp --acl public-read $(GOPATH)/bin/journalbeatlite s3://binaries-and-debs/bin/linux/journalbeatlite/
	aws s3 cp --acl public-read ./README.md s3://binaries-and-debs/bin/linux/journalbeatlite/
