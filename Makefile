.PHONY: test statik
test: statik
	go test ./... -v -count 1 -p 1 -cover
statik:
	go get github.com/rakyll/statik
	statik -src=template
	cd protoc-gen-lile-server && statik -src=template
clile: statik
	cd lile && go build -o /usr/local/bin/lile
cproto: statik
	cd protoc-gen-lile-server && go build -o /usr/local/bin/protoc-gen-lile-server
default: test
