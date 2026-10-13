# Local build targets. Every build and test runs without cgo.
export CGO_ENABLED := 0

BINARY    := pokemon-simulator
DIST      := dist
LDFLAGS   := -s -w
PLATFORMS := linux/amd64 linux/arm64 windows/amd64 darwin/amd64 darwin/arm64

# VERSION names the release archives and goes into the Windows build's
# properties: the latest v* tag as git describe gives it, or v0.0.0 and the
# commit while there is no tag.
DESCRIBE := $(shell git describe --tags --match 'v[0-9]*' --always --dirty 2>/dev/null)
VERSION  ?= $(if $(filter v%,$(DESCRIBE)),$(DESCRIBE),v0.0.0-$(or $(DESCRIBE),unknown))

# WINRES makes the Windows build's icon, manifest and properties from winres/.
WINRES := go run github.com/tc-hib/go-winres@v0.3.3

# LINT checks the code with the linters and formatters in .golangci.yaml.
LINT := golangci-lint run --config .golangci.yaml

# DOCS go into every archive: the licences, and the sources of the embedded
# fonts and icons (assets/README.md).
DOCS := README.md LICENSE assets/README.md assets/wine/LGPL-2.1.txt assets/wine/ms_sans_serif.sfd \
	$(wildcard assets/wine/*.svg) assets/liberation/OFL-1.1.txt

goos    = $(word 1,$(subst /, ,$1))
goarch  = $(word 2,$(subst /, ,$1))
windows = $(filter windows,$(call goos,$1))
exe     = $(if $(call windows,$1),.exe)
gui     = $(if $(call windows,$1),-H windowsgui)
bin     = $(DIST)/$(BINARY)-$(call goos,$1)-$(call goarch,$1)$(call exe,$1)
# pkg names a platform's archive, with macos for darwin.
pkg     = $(BINARY)-$(VERSION)-$(subst darwin,macos,$(call goos,$1))-$(call goarch,$1)

# package puts platform $1's binary in an archive, in a folder with DOCS:
# a .tar.gz for Linux, a .zip for the others.
define package
	rm -rf $(DIST)/$(call pkg,$1)
	mkdir -p $(DIST)/$(call pkg,$1)
	cp $(call bin,$1) $(DIST)/$(call pkg,$1)/$(BINARY)$(call exe,$1)
	tar -cf - $(DOCS) | tar -xf - -C $(DIST)/$(call pkg,$1)
	cd $(DIST) && $(if $(filter linux,$(call goos,$1)),\
		tar --owner=0 --group=0 --numeric-owner -czf $(call pkg,$1).tar.gz,\
		zip -qrX $(call pkg,$1).zip) $(call pkg,$1)
	rm -rf $(DIST)/$(call pkg,$1)

endef

.PHONY: run test vet lint build dist web serve clean $(PLATFORMS)

run:
	go run .

test:
	go test ./...

vet:
	go vet ./...

# lint checks every platform's code, as the build tags give Windows and the
# web files of their own; the web's tests build only in internal/platform.
lint:
	GOOS=linux $(LINT)
	GOOS=windows $(LINT)
	GOOS=js GOARCH=wasm $(LINT) --tests=false
	GOOS=js GOARCH=wasm $(LINT) ./internal/platform/

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY) .

# dist builds every platform and the web, and packs each into an archive in
# $(DIST). The web's has its files and DOCS at the top, as itch.io wants.
dist: $(PLATFORMS) web
	rm -f $(DIST)/*.tar.gz $(DIST)/*.zip
	$(foreach p,$(PLATFORMS),$(call package,$p))
	rm -rf $(DIST)/$(BINARY)-$(VERSION)-web
	cp -R $(DIST)/web $(DIST)/$(BINARY)-$(VERSION)-web
	tar -cf - $(DOCS) | tar -xf - -C $(DIST)/$(BINARY)-$(VERSION)-web
	cd $(DIST)/$(BINARY)-$(VERSION)-web && zip -qrX ../$(BINARY)-$(VERSION)-web.zip .
	rm -rf $(DIST)/$(BINARY)-$(VERSION)-web

$(PLATFORMS):
	$(if $(call windows,$@),$(WINRES) make --in winres/winres.json --arch $(call goarch,$@) \
		--product-version $(VERSION) --file-version $(VERSION))
	GOOS=$(call goos,$@) GOARCH=$(call goarch,$@) go build -trimpath \
		-ldflags "$(LDFLAGS) $(call gui,$@)" -o $(call bin,$@) .
	$(if $(call windows,$@),rm -f rsrc_windows_$(call goarch,$@).syso)

web:
	mkdir -p $(DIST)/web
	GOOS=js GOARCH=wasm go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/web/$(BINARY).wasm .
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" web/index.html $(DIST)/web/
	cp assets/sprites/icon.png $(DIST)/web/favicon.png

serve: web
	@echo "Serving $(DIST)/web at http://localhost:8080"
	python3 -m http.server --directory $(DIST)/web 8080

clean:
	rm -rf $(DIST)
