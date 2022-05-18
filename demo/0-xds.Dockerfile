FROM scratch
COPY example/bin/xds /bin/xds
ENTRYPOINT [ "/bin/xds" ]