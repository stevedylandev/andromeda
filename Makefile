GO_APPS := $(shell find apps -maxdepth 1 -type d -name '*-go' | sort)

.PHONY: help go-test go-vet go-fmt go-check go-app-test go-app-vet go-app-fmt

help:
	@echo "Available targets:"
	@echo "  make go-test              Run go test ./... in every apps/*-go module"
	@echo "  make go-vet               Run go vet ./... in every apps/*-go module"
	@echo "  make go-fmt               Run gofmt -w . in every apps/*-go module"
	@echo "  make go-check             Run go-fmt, go-test, and go-vet for all Go apps"
	@echo "  make go-app-test APP=...  Run go test ./... in one Go app, e.g. APP=feeds-go"
	@echo "  make go-app-vet APP=...   Run go vet ./... in one Go app"
	@echo "  make go-app-fmt APP=...   Run gofmt -w . in one Go app"

ifndef APP
go-app-test go-app-vet go-app-fmt:
	@echo "APP is required, e.g. make $@ APP=feeds-go" >&2
	@exit 1
else
go-app-test:
	@echo "==> apps/$(APP)"
	@cd apps/$(APP) && go test ./...

go-app-vet:
	@echo "==> apps/$(APP)"
	@cd apps/$(APP) && go vet ./...

go-app-fmt:
	@echo "==> apps/$(APP)"
	@cd apps/$(APP) && gofmt -w .
endif

go-test:
	@for app in $(GO_APPS); do \
		echo "==> $$app"; \
		(cd "$$app" && go test ./...) || exit $$?; \
	done

go-vet:
	@for app in $(GO_APPS); do \
		echo "==> $$app"; \
		(cd "$$app" && go vet ./...) || exit $$?; \
	done

go-fmt:
	@for app in $(GO_APPS); do \
		echo "==> $$app"; \
		(cd "$$app" && gofmt -w .) || exit $$?; \
	done

go-check: go-fmt go-test go-vet
