buildApprove:
	@mkdir -p build
	@go build -trimpath -ldflags="-s -w" -o build/annotation_approve approveGen/main.go
	@echo "success"

buildSwag:
	@mkdir -p build
	@go build -o build/swagGen ./swagGen
	@echo "success"

buildSetter:
	@mkdir -p build
	@go build -trimpath -ldflags="-s -w" -o build/setterGen ./setterGen
	@echo "success"

testSwag:
		@swag init --ot go \
    	--instanceName 'Test' \
    	--parseDependency \
    	--parseInternal \
    	-d ./ \
    	-g ./swagGen/swag-doc.go \
    	-o ./swagGen/docs/