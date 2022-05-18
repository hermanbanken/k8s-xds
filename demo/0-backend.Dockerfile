FROM scratch
COPY example/bin/server /bin/server
ENTRYPOINT [ "/bin/server" ]