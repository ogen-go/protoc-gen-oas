generate-proxy:
	go install -v ../cmd/protoc-gen-oas
	mkdir -pv ./proxypb
	protoc -I . \
			--oas_out=proxypb --oas_opt=proxy=true --oas_opt=paths=source_relative \
			--go_out=proxypb --go_opt=paths=source_relative \
			--go-grpc_out=proxypb --go-grpc_opt=paths=source_relative \
			proxy.proto
.PHONY: generate-proxy
