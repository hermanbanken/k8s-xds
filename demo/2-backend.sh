GOOS=linux GOARCH=arm64 go build -o ../example/bin/server ../example/server
docker build --platform=linux/arm64 -t demo-server -f 2-backend.Dockerfile ..
IMAGE=demo-server

## Without Apple M1 Silicon:
# kind --name=demo load docker-image demo-server
# docker exec -it $(docker ps) crictl images
# kubectl create deployment demo-server --image demo-server --replicas=1

## With Apple M1 Silicon:
docker tag demo-server localhost:5001/demo-server
docker push localhost:5001/demo-server
IMAGE=localhost:5001/demo-server

# Both:
kubectl create deployment demo-server --image $IMAGE --replicas=1
kubectl expose deployment demo-server --port=9090 --target-port=9090 --selector='app=demo-server' --cluster-ip='None'
kubectl set env deployment/demo-server JAEGER_TRACE_URL=http://localhost:14268/api/traces
