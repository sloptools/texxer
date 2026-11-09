WAILS ?= $(shell command -v wails 2>/dev/null || echo $(HOME)/go/bin/wails)
APP := texxer.app
DIST := dist
ADDON := tools/blender/texxer_bridge.py

.PHONY: build dev install dist dist-mac dist-windows dist-linux deps test clean

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

# One archive per platform in dist/, each with the app and the Blender add-on.
# Linux builds only on Linux (cgo, WebKitGTK 4.1), so dist leaves it out.
dist: dist-mac dist-windows
	@ls -lh $(DIST)

define stage
	rm -rf $(DIST)/$(1) && mkdir -p $(DIST)/$(1)
	cp -R build/bin/$(2) $(ADDON) $(DIST)/$(1)/
endef

dist-mac: deps
	$(WAILS) build -clean -platform darwin/universal
	$(call stage,mac,$(APP))
	cd $(DIST)/mac && zip -qry ../texxer-mac.zip .
	rm -rf $(DIST)/mac

dist-windows: deps
	$(WAILS) build -clean -platform windows/amd64
	$(call stage,windows,texxer.exe)
	cd $(DIST)/windows && zip -qr ../texxer-windows.zip .
	rm -rf $(DIST)/windows

dist-linux: deps
	$(WAILS) build -clean -platform linux/amd64 -tags webkit2_41
	$(call stage,linux,texxer)
	tar -czf $(DIST)/texxer-linux.tar.gz -C $(DIST)/linux .
	rm -rf $(DIST)/linux

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
