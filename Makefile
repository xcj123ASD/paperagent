.PHONY: ci fmt vet build test

ci: fmt-check vet build test

fmt:
gofmt -w .

fmt-check:
@unformatted=$$(gofmt -l .); \
if [ -n "$$unformatted" ]; then \
echo "以下文件未格式化："; \
echo "$$unformatted"; \
exit 1; \
fi

vet:
go vet ./...

build:
go build ./...

test:
go test ./...