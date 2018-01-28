# please exec before: Set-ExecutionPolicy RemoteSigned -Scope CurrentUser

$Protoc_java = $env:PROTOC_GEN_GRPC_JAVA

echo("Generating Go Protoc...")
protoc --go_out=plugins=grpc:. *.proto

echo("Generating Java Protoc...")
protoc --java_out=. --plugin=$Protoc_java=protoc-gen-grpc-java --grpc-java_out=. *.proto