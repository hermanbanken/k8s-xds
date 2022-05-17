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
kubectl create deployment demo-server --image="$IMAGE" --replicas=1
kubectl expose deployment demo-server --port=9090 --target-port=9090 --selector='app=demo-server' --cluster-ip='None'
kubectl set env deployment/demo-server JAEGER_TRACE_URL=http://jaeger:14268/api/traces

# Locally:
# docker run -d --name jaeger \
#   -e COLLECTOR_ZIPKIN_HOST_PORT=:9411 \
#   -p 5775:5775/udp \
#   -p 6831:6831/udp \
#   -p 6832:6832/udp \
#   -p 5778:5778 \
#   -p 16686:16686 \
#   -p 14250:14250 \
#   -p 14268:14268 \
#   -p 14269:14269 \
#   -p 9411:9411 \
#   jaegertracing/all-in-one:1.33