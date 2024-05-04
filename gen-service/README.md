## gen-service

Backend gRPC service

##### Setup environment

```
python3 -m venv venv
source venv/bin/activate
python -m pip install requirements.txt
```

##### Run
```
python -m main.py
```

##### Compile protocol buffers

```
python -m pip install dev-requirements.txt
python -m grpc_tools.protoc -I. --python_out=./grpc/ --pyi_out=./grpc/ --grpc_python_out=./grpc/ gen-service.proto
```

