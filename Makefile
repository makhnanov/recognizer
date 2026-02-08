.PHONY: run build clean install help

run: ## Build and run the application
	make install && go build -o recognizer recognizer.go && ./recognizer

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-10s %s\n", $$1, $$2}'

build: ## Compile the binary
	go build -o recognizer recognizer.go

install: ## Install dependencies (go mod tidy)
	go mod tidy

clean: ## Remove binary and recorded audio files
	rm -f recognizer
	rm -rf .voices/*.wav
