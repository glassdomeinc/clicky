ALL_GO_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | sort)

.PHONY: go_mod_tidy
go_mod_tidy:
	set -e; for dir in $(ALL_GO_MOD_DIRS); do \
	  echo "go mod tidy in $${dir}"; \
	  (cd "$${dir}" && go mod tidy); \
	done

.PHONY: deps
deps:
	set -e; for dir in $(ALL_GO_MOD_DIRS); do \
	  echo "go get -u ./... && go mod tidy in $${dir}"; \
	  (cd "$${dir}" && \
	    go get -u ./... && \
	    go mod tidy); \
	done

fmt:
	gofmt -w -s ./
	goimports -w  -local github.com/glassdomeinc/clicky ./

codegen:
	go run ./ch/internal/codegen/ -dir=ch/chschema
