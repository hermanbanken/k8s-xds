Buf.build usage
---------------
See: https://docs.buf.build/generate-usage/

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
export PATH=$HOME/go/bin:$PATH
brew install bufbuild/buf/buf
buf generate
```