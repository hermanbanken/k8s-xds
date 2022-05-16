GOOS=linux GOARCH=arm64 go build -o ../example/bin/client ../example/client
docker build --platform=linux/arm64 -t demo-client -f 2-client.Dockerfile ..
IMAGE=demo-client
TAG=$(git rev-parse --short head)

## Without Apple M1 Silicon:
# kind --name=demo load docker-image demo-client
# docker exec -it $(docker ps) crictl images
# kubectl create deployment demo-client --image demo-client:$TAG --replicas=1

## With Apple M1 Silicon:
TAG=$(git rev-parse --short head)
docker tag demo-client localhost:5001/demo-client:$TAG
docker push localhost:5001/demo-client:$TAG
IMAGE=localhost:5001/demo-client:$TAG

# Both:
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: demo-client
  name: demo-client
spec:
  replicas: 5
  selector:
    matchLabels:
      app: demo-client
  template:
    metadata:
      labels:
        app: demo-client
    spec:
      containers:
      - image: $IMAGE
        name: demo-client
        env:
        - name: JAEGER_TRACE_URL
          value: http://jaeger:14268/api/traces
        - name: UPSTREAM_HOST
          value: demo-server:9090
        - name: DURATION
          value: 500ms
        readinessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 1
          periodSeconds: 1
          failureThreshold: 1
EOF
kubectl get service demo-client || kubectl expose deployment demo-client --port=9090 --target-port=9090 --selector='app=demo-client' --cluster-ip='None'
