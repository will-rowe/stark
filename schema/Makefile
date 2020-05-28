# from: https://blog.gopheracademy.com/advent-2017/make/
.PHONY: compile
PROTOC_GEN_GO := $(GOPATH)/bin/protoc-gen-go
PROTOC := $(shell which protoc)
# If protoc isn't on the path, set it to a target that's never up to date, so
# the install command always runs.
ifeq ($(PROTOC),)
    PROTOC = must-rebuild
endif

# Figure out which machine we're running on.
UNAME := $(shell uname)

$(PROTOC):
# Run the right installation command for the operating system.
ifeq ($(UNAME), Darwin)
	brew install protobuf
endif
ifeq ($(UNAME), Linux)
	sudo apt-get install protobuf-compiler
endif


# If $GOPATH/bin/protoc-gen-go does not exist, we'll run this command to install
# it.
$(PROTOC_GEN_GO):
	go get -u github.com/golang/protobuf/protoc-gen-go

../src/stark.pb.go: protobuf/stark.proto | $(PROTOC_GEN_GO) $(PROTOC)
	protoc -I=protobuf/ --go_out=../src protobuf/stark.proto

# This is a "phony" target - an alias for the above command, so "make compile"
# still works.
compile: ../src/stark.pb.go