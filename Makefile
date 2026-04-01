UNAME := $(shell uname -s)
CODESIGN_IDENTITY ?= -
CODESIGN_IDENTIFIER ?= tech.heyworth.sympa

install:
	-sympa agent stop 2>/dev/null
	go build -o ./sympa ./cmd/sympa
ifeq ($(UNAME),Darwin)
	codesign -s "$(CODESIGN_IDENTITY)" --identifier "$(CODESIGN_IDENTIFIER)" ./sympa
endif
	sudo cp ./sympa /usr/local/bin/sympa
	@rm -f ./sympa

release:
	@if grep -q '^replace' go.mod; then \
		echo "ERROR: go.mod contains replace directive — remove it before releasing"; \
		exit 1; \
	fi
	@echo "Ready to tag. Run: git tag v0.x.x && git push origin v0.x.x"
