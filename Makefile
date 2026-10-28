# Local build targets. Every build and test runs without cgo.
export CGO_ENABLED := 0

BINARY    := pokemon-simulator
DIST      := dist
PLATFORMS := linux/amd64 linux/arm64 windows/amd64 darwin/amd64 darwin/arm64

# VERSION names the release archives, goes into the Windows build's
# properties and into the program, which shows it: the latest v* tag as git
# describe gives it, or v0.0.0 and the commit while there is no tag.
DESCRIBE := $(shell git describe --tags --match 'v[0-9]*' --always --dirty 2>/dev/null)
VERSION  ?= $(if $(filter v%,$(DESCRIBE)),$(DESCRIBE),v0.0.0-$(or $(DESCRIBE),unknown))

# LDFLAGS strips the binaries and gives the program its VERSION.
LDFLAGS := -s -w -X github.com/jsperandio/pokemon-simulator/internal/version.tag=$(VERSION)

# WINRES makes the Windows build's icon, manifest and properties from winres/.
WINRES := go run github.com/tc-hib/go-winres@v0.3.3

# LINT checks the code with the linters and formatters in .golangci.yaml.
LINT := golangci-lint run --config .golangci.yaml

# DOCS go into every archive: the licences, the sources of the embedded
# fonts and icons (assets/README.md), and the pages of docs/ the README links.
DOCS := README.md docs/improvements.md docs/playing.md LICENSE assets/README.md assets/wine/LGPL-2.1.txt assets/wine/ms_sans_serif.sfd \
	$(wildcard assets/wine/*.svg) assets/liberation/OFL-1.1.txt

# LINUX_FILES go into the Linux archives: the desktop entry and the scripts that install and remove it.
LINUX_FILES := packaging/linux/pokemon-simulator.desktop packaging/linux/install-desktop.sh packaging/linux/uninstall-desktop.sh

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
	$(if $(filter linux,$(call goos,$1)),cp $(LINUX_FILES) $(DIST)/$(call pkg,$1)/ && cp assets/sprites/icon.png $(DIST)/$(call pkg,$1)/pokemon-simulator.png)
	cd $(DIST) && $(if $(filter linux,$(call goos,$1)),\
		tar --owner=0 --group=0 --numeric-owner -czf $(call pkg,$1).tar.gz,\
		zip -qrX $(call pkg,$1).zip) $(call pkg,$1)
	rm -rf $(DIST)/$(call pkg,$1)

endef

.PHONY: run test vet lint build dist web serve clean $(PLATFORMS)

run:
	go run -ldflags "$(LDFLAGS)" .

test:
	go test ./...

vet:
	go vet ./...

# lint checks every platform's code, as the build tags give Windows and the
# web files of their own.
lint:
	GOOS=linux $(LINT)
	GOOS=windows $(LINT)
	GOOS=js GOARCH=wasm $(LINT)

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY) .

# dist builds every platform and the web, and packs each into an archive in
# $(DIST). The web's has its files and DOCS at the top, with no folder.
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
