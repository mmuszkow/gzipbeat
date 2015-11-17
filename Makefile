PREFIX?=/build

GOFILES = $(shell find . -type f -name '*.go')
gzipbeat: $(GOFILES)
	go build

clean:
	rm gzipbeat || true

