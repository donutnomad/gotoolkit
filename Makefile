buildApprove:
	@mkdir -p build
	@go build -trimpath -ldflags="-s -w" -o build/annotation_approve approveGen/main.go
	@echo "success"

buildSwag:
	@mkdir -p build
	@go build -o build/swagGen ./swagGen
	@echo "success"
