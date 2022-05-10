APP := best_practicies
PROJECT := https://github.com/mswatermelon/GB_march_best_practic

.PHONY: install-tools
install-tools:
	if [ ! -f goimports ]; then \
		go install golang.org/x/tools/cmd/goimports@latest; \
	fi;
	if [ ! -f golangci-lint ]; then \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.45.2; \
	fi;

.PHONY: check
check: install-tools
	$(go env GOPATH)/bin/golangci-lint run ./...

.PHONY: test
test:
	go test -race ./...

.PHONY: build
build:
	go build -a  -o $(APP) $(PROJECT)

