# Local build targets. Every build and test runs without cgo.
export CGO_ENABLED := 0

BINARY    := pokemon-simulator
DIST      := dist
LDFLAGS   := -s -w
PLATFORMS := linux/amd64 linux/arm64 windows/amd64 darwin/amd64 darwin/arm64

# VERSION goes into the Windows build's properties: the latest v* tag as git
# describe gives it, or v0.0.0 and the commit while there is no tag.
DESCRIBE := $(shell git describe --tags --match 'v[0-9]*' --always --dirty 2>/dev/null)
VERSION  ?= $(if $(filter v%,$(DESCRIBE)),$(DESCRIBE),v0.0.0-$(or $(DESCRIBE),unknown))

# WINRES makes the Windows build's icon, manifest and properties from winres/.
WINRES := go run github.com/tc-hib/go-winres@v0.3.3

goos    = $(word 1,$(subst /, ,$1))
goarch  = $(word 2,$(subst /, ,$1))
windows = $(filter windows,$(call goos,$1))
exe     = $(if $(call windows,$1),.exe)
gui     = $(if $(call windows,$1),-H windowsgui)

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
	$(if $(call windows,$@),$(WINRES) make --in winres/winres.json --arch $(call goarch,$@) \
		--product-version $(VERSION) --file-version $(VERSION))
	GOOS=$(call goos,$@) GOARCH=$(call goarch,$@) go build -trimpath \
		-ldflags "$(LDFLAGS) $(call gui,$@)" \
		-o $(DIST)/$(BINARY)-$(call goos,$@)-$(call goarch,$@)$(call exe,$@) .
	$(if $(call windows,$@),rm -f rsrc_windows_$(call goarch,$@).syso)

web:
	mkdir -p $(DIST)/web
	GOOS=js GOARCH=wasm go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/web/$(BINARY).wasm .
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" web/index.html $(DIST)/web/

serve: web
	@echo "Serving $(DIST)/web at http://localhost:8080"
	python3 -m http.server --directory $(DIST)/web 8080

clean:
	rm -rf $(DIST)
