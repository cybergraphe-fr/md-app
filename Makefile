.PHONY: dev dev-frontend build test test-race test-frontend lint vet vuln check clean docker docker-nas ci \
	desktop-install-tools desktop-bin-win-x64 desktop-bin-macos-amd64 desktop-bin-macos-arm64 \
	desktop-bin-all desktop-package-win-x64 desktop-package-macos desktop-package-all

dev:
	go run ./cmd/server

dev-frontend:
	cd web && npm run dev

build:
	go build -o build/md ./cmd/server

test:
	go test ./...

test-race:
	go test -race -timeout 120s ./...

test-frontend:
	cd web && npm test

lint:
	golangci-lint run

vet:
	go vet ./...

vuln:
	govulncheck ./...

check:
	cd web && npm run check

clean:
	rm -rf build/ web/dist/ coverage/

docker:
	docker compose up --build

docker-nas:
	docker compose -f docker-compose.nas.yml up -d --build

ci: vet lint test check test-frontend

desktop-install-tools:
	go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2

desktop-bin-win-x64:
	mkdir -p build/desktop/windows-x64
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags desktop -o build/desktop/windows-x64/md-desktop.exe ./cmd/desktop

desktop-bin-macos-amd64:
	mkdir -p build/desktop/macos
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -tags desktop -o build/desktop/macos/md-desktop-amd64 ./cmd/desktop

desktop-bin-macos-arm64:
	mkdir -p build/desktop/macos
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -tags desktop -o build/desktop/macos/md-desktop-arm64 ./cmd/desktop

desktop-bin-all: desktop-bin-win-x64 desktop-bin-macos-amd64 desktop-bin-macos-arm64

desktop-package-win-x64:
	bash desktop/windows-x64/scripts/build-win-x64.sh

desktop-package-macos:
	bash desktop/macos/scripts/build-macos.sh

desktop-package-all: desktop-package-win-x64 desktop-package-macos
