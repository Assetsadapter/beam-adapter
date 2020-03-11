.PHONY: all clean
.PHONY: openw-beam
.PHONY: deps

# Check for required command tools to build or stop immediately
EXECUTABLES = git go find pwd
K := $(foreach exec,$(EXECUTABLES),\
        $(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH)))

GO ?= latest

# openw-beam
BEAMWALLETVERSION = $(shell git describe --tags `git rev-list --tags --max-count=1`)
BEAMWALLETBINARY = openw-beam
BEAMWALLETMAIN = main.go

BUILDDIR = build
GITREV = $(shell git rev-parse --short HEAD)
BUILDTIME = $(shell date +'%Y-%m-%d_%T')

BEAMWALLETLDFLAGS="-X github.com/blocktree/beam-adapter/commands.Version=${BEAMWALLETVERSION} \
	-X github.com/blocktree/beam-adapter/commands.GitRev=${GITREV} \
	-X github.com/blocktree/beam-adapter/commands.BuildTime=${BUILDTIME}"

# OS platfom
# options: windows-6.0/*,darwin-10.10/amd64,linux/amd64,linux/386,linux/arm64,linux/mips64, linux/mips64le
TARGETS="darwin-10.10/amd64,linux/amd64"

deps:
	go get -u github.com/gythialy/xgo

build:
	GO111MODULE=on go build -ldflags $(BEAMWALLETLDFLAGS) -i -o $(shell pwd)/$(BUILDDIR)/$(BEAMWALLETBINARY) $(shell pwd)/$(BEAMWALLETMAIN)
	@echo "Build $(BEAMWALLETBINARY) done."

all: openw-beam

clean:
	rm -rf $(shell pwd)/$(BUILDDIR)/

openw-beam:
	xgo --dest=$(BUILDDIR) --ldflags=$(BEAMWALLETLDFLAGS) --out=$(BEAMWALLETBINARY)-$(BEAMWALLETVERSION)-$(GITREV) --targets=$(TARGETS) \
	--pkg=$(BEAMWALLETMAIN) .
