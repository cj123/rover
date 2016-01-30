all: build

GIT_VERSION := $(shell git rev-parse --short HEAD)

build: $(wildcard *.go)
	GOOS=linux  GOARCH=amd64 go build -ldflags -w -o build/rover-linux-x64
	GOOS=darwin GOARCH=amd64 go build -ldflags -w -o build/rover-osx-x64
	GOOS=windows GOARCH=386 go build -ldflags -w -o build/rover-windows.exe

archive: build
	cp README.md build
	cd build
	zip rover-win-$(GIT_VERSION).zip build/rover-windows.exe README.md
	zip rover-osx-$(GIT_VERSION).zip build/rover-osx-x64 README.md
	zip rover-lin-$(GIT_VERSION).zip build/rover-linux-x64 README.md
	cd ..

clean:
	rm -rf build
