.PHONY: all fmt document check

all: fmt document

document:
	go run github.com/codectl/markparsr/cmd/markparsr@latest generate .

check:
	go run github.com/codectl/markparsr/cmd/markparsr@latest check .

fmt:
	terraform fmt -recursive
