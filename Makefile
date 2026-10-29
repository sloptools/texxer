.PHONY: run build web test dev dev-server dev-web

# Builds everything and starts the editor on http://localhost:7777.
run: build
	./texxer serve -dir examples

build: web
	go build -o texxer ./cmd/texxer

web: web/node_modules
	cd web && npm run build

web/node_modules: web/package.json web/package-lock.json
	cd web && npm install
	touch web/node_modules

test: web/node_modules
	go vet ./...
	go test ./...
	cd web && npm run check

# Go server plus Vite with hot reload on http://localhost:5173.
dev: web/node_modules
	$(MAKE) -j2 dev-server dev-web

dev-server:
	go run ./cmd/texxer serve -dir examples

dev-web:
	cd web && npm run dev
