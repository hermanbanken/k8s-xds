GOOS=linux GOARCH=arm64 go build -o ../example/bin/client ../example/client
docker build --platform=linux/arm64 -t demo-client -f 2-client.Dockerfile ..
IMAGE=demo-client

## Without Apple M1 Silicon:
# kind --name=demo load docker-image demo-client
# docker exec -it $(docker ps) crictl images
# kubectl create deployment demo-client --image demo-client --replicas=1

## With Apple M1 Silicon:
docker tag demo-client localhost:5001/demo-client
docker push localhost:5001/demo-client
IMAGE=localhost:5001/demo-client

# Both:
kubectl create deployment demo-client --image="$IMAGE" --replicas=5
kubectl expose deployment demo-client --port=9090 --target-port=9090 --selector='app=demo-client' --cluster-ip='None'
kubectl set env deployment/demo-client JAEGER_TRACE_URL=http://jaeger:14268/api/traces UPSTREAM_HOST=demo-server:9090 DURATION=10ms