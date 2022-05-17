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
  replicas: 2
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
          value: demo-server-headless:9090
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

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: demo-server-clusterip
spec:
  type: ClusterIP
  selector: { app: demo-server }
  ports:
  - port: 9090
    protocol: TCP
    targetPort: 9090
EOF

kubectl set env deployment/demo-client UPSTREAM_HOST=demo-server-clusterip:9090
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: demo-server-headless
spec:
  type: ClusterIP
  clusterIP: None
  selector: { app: demo-server }
  ports:
  - port: 9090
    protocol: TCP
    targetPort: 9090
EOF