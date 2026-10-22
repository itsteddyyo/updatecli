build:
	GOOS=wasip1 GOARCH=wasm go build -tags std -o argocd-valuesobject.wasm .

test: build
	go test ./...