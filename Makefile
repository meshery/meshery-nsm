protoc-setup:
	cd meshes
	wget https://raw.githubusercontent.com/layer5io/meshery/master/meshes/meshops.proto

proto:	
	protoc -I meshes/ meshes/meshops.proto --go_out=plugins=grpc:./meshes/

docker:
	docker build -t layer5/meshery-nsm .

docker-run:
	(docker rm -f meshery-nsm) || true
	docker run --name meshery-nsm -d \
	-p 10004:10004 \
	-e DEBUG=true \
	layer5/meshery-nsm

run:
	DEBUG=true go run main.go
