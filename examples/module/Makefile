.PHONY: all fmt document check

all: fmt document

document:
	go run github.com/codectl/shapr/cmd/shapr generate .

check:
	go run github.com/codectl/shapr/cmd/shapr check .

fmt:
	terraform fmt -recursive
