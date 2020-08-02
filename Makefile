ifeq ($(OS),Windows_NT)
	builddate = $(shell echo %date:~0,4%%date:~5,2%%date:~8,2%%time:~0,2%%time:~3,2%%time:~6,2%)
else
	builddate = $(shell date +"%Y-%M-%d %H:%M:%S")
endif

version = v0.1.6
ldflags = -X 'main.version=$(version)' -X 'main.builddate=$(builddate)'

build:
	clean
	go build -o build/wickproxy .

clean:
	rm -f build/wickproxy*

linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64   go build -ldflags "$(ldflags)" -o build/wickproxy-linux-amd64-$(version) .
linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64   go build -ldflags "$(ldflags)" -o build/wickproxy-linux-arm64-$(version) .
darwin-amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64  go build -ldflags "$(ldflags)" -o build/wickproxy-darwin-amd64-$(version) .
windows-x64:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(ldflags)" -o build/wickproxy-windows-x64-$(version).exe .
windows-x86:
	CGO_ENABLED=0 GOOS=windows GOARCH=386   go build -ldflags "$(ldflags)" -o build/wickproxy-windows-x86-$(version).exe .
freebsd-amd64:
	CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -ldflags "$(ldflags)" -o build/wickproxy-freebsd-amd64-$(version) .

all: clean linux-amd64 linux-arm64 darwin-amd64 windows-x64 windows-x86 freebsd-amd64