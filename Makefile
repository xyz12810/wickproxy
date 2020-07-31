build:
	clean
	go build -o build/wickproxy .

clean:
	rm build/wickproxy*

linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/wickproxy-linux-amd64 .
windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o build/wickproxy-windows-amd64.exe .

darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o build/wickproxy-darwin-amd64 .

all: linux darwin windows