FROM scratch
COPY example/bin/client /bin/client
COPY demo/3-bootstrap.json /bootstrap.json
ENV GRPC_XDS_BOOTSTRAP /bootstrap.json
ENTRYPOINT [ "/bin/client" ]