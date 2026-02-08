.PHONY: run build clean install help

run: ## Build and run the application
	make install && go build -o recognize recognize.go && ./recognize

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-10s %s\n", $$1, $$2}'

build: ## Compile the binary
	go build -o recognize recognize.go

install: ## Install dependencies (go mod tidy)
	go mod tidy

clean: ## Remove binary and recorded audio files
	rm -f recognize
	rm -rf .voices/*.wav
