```console
protoc -I . \
--oas_out=title="Generated OpenAPI specification from proto file",version=v1.0.0:. \
service.proto \
message.proto \
basic/v1/error.proto
```
