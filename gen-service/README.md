## gen-service

Backend gRPC service to generate audio effect from a prompt

##### Setup environment

```
python3 -m venv venv
source venv/bin/activate
#windows
.\venv\Scripts\Activate.ps1
python -m pip install -r requirements.txt
```

##### Run
```
python -m server
```

##### Compile python protobufs

TODO script to compile protobufs

```
python -m pip install -r dev-requirements.txt
python -m grpc_tools.protoc -I. --python_out=./ --pyi_out=./ --grpc_python_out=./ ./gen-service.proto
```

##### Compile go protobufs
```
brew install protobuf
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
export PATH="$PATH:$(go env GOPATH)/bin"
protoc --proto_path=. --go_out=. --go_opt=paths=source_relative gen-service.proto
```
