SHELL := /bin/bash

HASH := $(shell git rev-parse --short head)
DATE := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

.PHONY: install
install:
	-rm /home/ubuntu/go/bin/journalbeatlite
	go install -ldflags '-X main.BuildHash=$(HASH) -X main.BuildDate=$(DATE)' .

.PHONY: upload
upload: install
	aws s3 cp --acl public-read $(GOPATH)/bin/journalbeatlite s3://binaries-and-debs/bin/linux/journalbeatlite/
	aws s3 cp --acl public-read ./README.md s3://binaries-and-debs/bin/linux/journalbeatlite/
