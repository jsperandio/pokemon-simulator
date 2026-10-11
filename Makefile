# Local build targets. Every build and test runs without cgo.
export CGO_ENABLED := 0

BINARY    := pokemon-simulator
DIST      := dist
LDFLAGS   := -s -w
PLATFORMS := linux/amd64 linux/arm64 windows/amd64 darwin/amd64 darwin/arm64

goos   = $(word 1,$(subst /, ,$1))
goarch = $(word 2,$(subst /, ,$1))
exe    = $(if $(filter windows,$(call goos,$1)),.exe)
gui    = $(if $(filter windows,$(call goos,$1)),-H windowsgui)

.PHONY: run test vet build dist web serve clean $(PLATFORMS)

run:
	go run .

test:
	go test ./...

vet:
	go vet ./...

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY) .

dist: $(PLATFORMS) web

$(PLATFORMS):
	GOOS=$(call goos,$@) GOARCH=$(call goarch,$@) go build -trimpath \
		-ldflags "$(LDFLAGS) $(call gui,$@)" \
		-o $(DIST)/$(BINARY)-$(call goos,$@)-$(call goarch,$@)$(call exe,$@) .

web:
	mkdir -p $(DIST)/web
	GOOS=js GOARCH=wasm go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/web/$(BINARY).wasm .
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" web/index.html $(DIST)/web/

serve: web
	@echo "Serving $(DIST)/web at http://localhost:8080"
	python3 -m http.server --directory $(DIST)/web 8080

clean:
	rm -rf $(DIST)
