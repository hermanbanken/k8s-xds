TAG=$(git rev-parse --short head)
TAG=latest

export GOOS=linux GOARCH=arm64
go build -o ../example/bin/xds ../
go build -o ../example/bin/server ../example/server
go build -o ../example/bin/client ../example/client

docker build --platform=linux/arm64 -t demo-xds -f 0-xds.Dockerfile ..
docker build --platform=linux/arm64 -t demo-server -f 0-backend.Dockerfile ..
docker build --platform=linux/arm64 -t demo-client -f 0-client.Dockerfile ..

docker tag demo-xds localhost:5001/demo-xds:$TAG
docker tag demo-server localhost:5001/demo-server:$TAG
docker tag demo-client localhost:5001/demo-client:$TAG

docker push localhost:5001/demo-xds:$TAG
docker push localhost:5001/demo-server:$TAG
docker push localhost:5001/demo-client:$TAG

# kustomize edit set image "demo-xds=localhost:5001/demo-xds:$TAG"
# kustomize edit set image "demo-server=localhost:5001/demo-server:$TAG"
# kustomize edit set image "demo-client=localhost:5001/demo-client:$TAG"