FROM scratch
COPY example/bin/client /bin/client
ENTRYPOINT [ "/bin/client" ]