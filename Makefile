WAILS ?= $(shell command -v wails 2>/dev/null || echo $(HOME)/go/bin/wails)
APP := texxer.app
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
DIST := dist
ADDON := tools/blender/texxer_bridge.py

.PHONY: build dev install dist deps test clean

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

# One zip per platform in dist/, each with the app and the Blender add-on.
dist: deps
	rm -rf $(DIST) && mkdir -p $(DIST)/mac $(DIST)/windows
	$(WAILS) build -clean -platform darwin/universal
	cp -R build/bin/$(APP) $(DIST)/mac/
	cp $(ADDON) $(DIST)/mac/
	cd $(DIST)/mac && zip -qry ../texxer-$(VERSION)-mac.zip .
	$(WAILS) build -clean -platform windows/amd64
	cp build/bin/texxer.exe $(DIST)/windows/
	cp $(ADDON) $(DIST)/windows/
	cd $(DIST)/windows && zip -qr ../texxer-$(VERSION)-windows.zip .
	rm -rf $(DIST)/mac $(DIST)/windows
	@ls -lh $(DIST)

# main.go embeds frontend/dist, which must not be empty before the first frontend build.
deps:
	@test -x $(WAILS) || go install github.com/wailsapp/wails/v2/cmd/wails@latest
	@test -d frontend/node_modules || (cd frontend && npm install)
	@mkdir -p frontend/dist && touch frontend/dist/.keep

test: deps
	go vet ./...
	go test ./...
	cd frontend && npm run check

clean:
	rm -rf bin build/bin frontend/dist $(DIST)
