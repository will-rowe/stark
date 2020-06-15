APP := "stark"
API := "stark.pb.go"

.PHONY: all api build_app clean

all: build_app

stark.pb.go: schema/stark.proto
	@protoc -I schema/ \
		--go_out=plugins=grpc:./ \
		schema/stark.proto

api: stark.pb.go

dep:
	@go get -v -d ./...

build_app: dep api
	@go build -v -o $(APP) app/stark.go

clean:
	@rm $(APP) $(API)
