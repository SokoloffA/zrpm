PROJECT = github.com/sokoloffA/zrpm
VERSION := $(shell awk -F'"' '/version =/ {print($$2)}' main.go)

# # Paths ..........................
DESTDIR     =
PREFIX      = /usr/local

#***********************************************************
# Build variables
SOURCES = $(shell find -name "*.go" -not -path "./vendor/*")
BINARY  ?= $(notdir $(PROJECT))
MAKEFILE_DIR ?= $(realpath $(dir $(lastword $(MAKEFILE_LIST))))
BUILD_DIR = $(MAKEFILE_DIR)/.build
BUILD_BINARY = $(BUILD_DIR)/bin/$(BINARY)
PROJECT_BASE = $(notdir $(PROJECT))

GO_ENV=GOPATH=$(BUILD_DIR) GO15VENDOREXPERIMENT=1
#***********************************************************

all: $(BUILD_BINARY)

$(BUILD_DIR)/src/$(PROJECT):
	@mkdir -p $(dir $@)
	@ln -s $(MAKEFILE_DIR) $@

$(BUILD_BINARY): $(BUILD_DIR)/src/$(PROJECT) $(SOURCES)
	@$(GO_ENV) go install $(PROJECT)
	@strip $(BUILD_BINARY)

install:all
	@install -d $(DESTDIR)$(PREFIX)
	@install -v -m 755 $(BUILD_BINARY) $(DESTDIR)$(PREFIX)/$(BINARY)

uninstall:
	@rm -v -f $(DESTDIR)$(PREFIX)/$(BINARY)

clean:
	@rm -rf $(MAKEFILE_DIR)/.build

copysrc: $(BUILD_DIR)/src/$(PROJECT)
	@rm -rf "$(BUILD_DIR)/$(PROJECT_BASE)-$(VERSION)"
	@install -d "$(BUILD_DIR)/$(PROJECT_BASE)-$(VERSION)"
	@cp -ra $(BUILD_DIR)/src/$(PROJECT)/* "$(BUILD_DIR)/$(PROJECT_BASE)-$(VERSION)"

srctar: copysrc 
	@tar -cj --directory="$(BUILD_DIR)" -f "$(BUILD_DIR)/$(PROJECT_BASE)-$(VERSION).tbz" "$(PROJECT_BASE)-$(VERSION)"
	@rm -rf "$(BUILD_DIR)/$(PROJECT_BASE)-$(VERSION)"
	@echo Tar "$(BUILD_DIR)/$(PROJECT_BASE)-$(VERSION).tbz" is ready

bintar: $(BUILD_BINARY)
	@tar -cj --directory="$(BUILD_DIR)/bin" -f "$(BUILD_DIR)/$(PROJECT_BASE)-$(VERSION).tbz" "$(BUILD_BINARY)"
