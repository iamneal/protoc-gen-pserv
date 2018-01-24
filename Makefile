all: deps build

build: generator protos

protos:
	protoc -I. -I$$GOPATH/src -I./test \
		--plugin=./protoc-gen-pserv \
		--pserv_out=. ./test/*.proto
	# protoc -I. -I$$GOPATH/src -I./test \
	# 	--persist_out=persist_root=github.com/iamneal/protoc-gen-pserv/test:. --go_out=plugins=grpc:. \
	# 	./test/*.proto

generator:
	go build .

deps:
	go get -u github.com/golang/protobuf/protoc-gen-go
	go get -u github.com/tcncloud/protoc-gen-persist

clean:
	rm -f ./protoc-gen-pserv
	rm -f test/*.generated.proto
	rm -f test/*.persist.go
	rm -f test/*.pb.go

test:
	go test ./test/...


	