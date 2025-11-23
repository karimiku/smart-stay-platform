# ==============================================================================
# Variables
# ==============================================================================
PROTO_DIR = proto
GENPROTO_DIR = pkg/genproto

# Find all .proto files in the directory
PROTO_FILES = $(wildcard $(PROTO_DIR)/*.proto)

# ==============================================================================
# Targets
# ==============================================================================

.PHONY: help
help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: proto
proto: ## Generate Go code from .proto files to pkg/genproto
	@echo "🚀 Generating gRPC code..."
	@for file in $(PROTO_FILES); do \
		SERVICE_NAME=$$(basename $$file .proto); \
		TARGET_DIR=$(GENPROTO_DIR)/$$SERVICE_NAME; \
		echo "  > Processing: $$SERVICE_NAME -> $$TARGET_DIR"; \
		mkdir -p $$TARGET_DIR; \
		protoc --proto_path=$(PROTO_DIR) \
			--go_out=$$TARGET_DIR --go_opt=paths=source_relative \
			--go-grpc_out=$$TARGET_DIR --go-grpc_opt=paths=source_relative \
			$$file; \
	done
	@echo "✅ Done!"

.PHONY: clean
clean: ## Clean generated proto files
	@echo "🧹 Cleaning generated proto files..."
	@rm -rf $(GENPROTO_DIR)/*
	@echo "✅ Done!"

.PHONY: install-tools
install-tools: ## Install necessary protobuf tools (Run once)
	@echo "🛠 Installing protoc plugins..."
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "✅ Tools installed!"