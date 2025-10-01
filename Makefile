.PHONY: run build clean install

run:
	go run recognize.go

build:
	go build -o recognize recognize.go

install:
	go mod tidy

clean:
	rm -f recognize
	rm -rf .voices/*.wav
