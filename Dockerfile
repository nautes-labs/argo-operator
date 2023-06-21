FROM harbor.bluzin.io/library/static:nonroot
WORKDIR /
COPY /bin/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
