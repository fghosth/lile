Version := beta
.PHONY: test tmpstatik
build: tmpstatik
	mkdir -p ./dist
	go build -ldflags "-X main.Version=$(Version) -X 'main.BuildTime=`date`' -X 'main.GoVersion=`go version`'"  -o ./dist/lile ./lile
	go build -ldflags "-X main.Version=$(Version) -X 'main.BuildTime=`date`' -X 'main.GoVersion=`go version`'"  -o ./dist/protoc-gen-lile-server ./protoc-gen-lile-server
	tar czvf lile.tar.gz -C ./dist .
	shasum -a 256 lile.tar.gz
test: tmpstatik
	go test ./... -v -count 1 -p 1 -cover
tmpstatik:
	#go get github.com/rakyll/statik
	statik -src=template
	cd protoc-gen-lile-server && statik -src=template
clile: tmpstatik
	cd lile && go build -ldflags "-X main.Version=$(Version) -X 'main.BuildTime=`date`' -X 'main.GoVersion=`go version`'"  -o /usr/local/bin/lile
cproto: tmpstatik
	cd protoc-gen-lile-server && go build -ldflags "-X main.Version=$(Version) -X 'main.BuildTime=`date`' -X 'main.GoVersion=`go version`'"  -o /usr/local/bin/protoc-gen-lile-server
default: test
