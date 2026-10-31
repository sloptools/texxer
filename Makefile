WAILS ?= $(shell command -v wails 2>/dev/null || echo $(HOME)/go/bin/wails)
APP := texxer.app

.PHONY: build dev install deps test docs clean

build: deps
	$(WAILS) build -clean
	mkdir -p bin
	rm -rf bin/$(APP)
	cp -R build/bin/$(APP) bin/

dev: deps
	$(WAILS) dev

install: build
	rm -rf /Applications/$(APP)
	cp -R bin/$(APP) /Applications/

# main.go embeds frontend/dist, which must not be empty before the first frontend build.
deps:
	@test -x $(WAILS) || go install github.com/wailsapp/wails/v2/cmd/wails@latest
	@test -d frontend/node_modules || (cd frontend && npm install)
	@mkdir -p frontend/dist && touch frontend/dist/.keep

test: deps
	go vet ./...
	go test ./...
	cd frontend && npm run check

# Screenshots for the docs, taken from the example ship in the running dev app (make dev).
docs:
	cd frontend && node scripts/docs-screens.mjs

clean:
	rm -rf bin build/bin frontend/dist
