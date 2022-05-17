FROM scratch
COPY example/bin/xds /bin/xds
COPY demo/3-app.yaml /app.yaml
ENV CONFIG=/app.yaml
ENTRYPOINT [ "/bin/xds" ]