# xDS
GOOS=linux GOARCH=arm64 go build -o ../example/bin/xds ../
docker build --platform=linux/arm64 -t demo-xds -f 3-xds.Dockerfile ..
docker tag demo-xds localhost:5001/demo-xds
docker push localhost:5001/demo-xds
IMAGE=localhost:5001/demo-xds

# xDS client
GOOS=linux GOARCH=arm64 go build -o ../example/bin/client ../
docker build --platform=linux/arm64 -t demo-client-xds -f 3-client.Dockerfile ..
docker tag demo-client-xds localhost:5001/demo-client-xds
docker push localhost:5001/demo-client-xds
IMAGE_XDS=localhost:5001/demo-client-xds

# Both:
kubectl create deployment demo-xds --image="$IMAGE" --replicas=1
kubectl expose deployment demo-xds --port=9000 --target-port=9000 --selector='app=demo-xds'

cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: demo-client-xds
  name: demo-client-xds
spec:
  replicas: 5
  selector:
    matchLabels:
      app: demo-client-xds
  template:
    metadata:
      labels:
        app: demo-client-xds
    spec:
      containers:
      - image: $IMAGE_XDS
        name: demo-client-xds
        env:
        - name: JAEGER_TRACE_URL
          value: http://jaeger:14268/api/traces
        - name: UPSTREAM_HOST
          value: demo-server-headless
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