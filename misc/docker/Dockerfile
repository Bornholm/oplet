FROM alpine:3.23 AS runtime

ARG TARGETPLATFORM

RUN apk add \
    ca-certificates \
    openssl \
    gcompat \
  && update-ca-certificates

COPY $TARGETPLATFORM/oplet /usr/local/bin/oplet

CMD ["/usr/local/bin/oplet"]