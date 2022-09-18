FROM alpine

RUN apk add e2fsprogs

COPY bsos /usr/local/bin/bsos

ENTRYPOINT ["/usr/local/bin/bsos"]
