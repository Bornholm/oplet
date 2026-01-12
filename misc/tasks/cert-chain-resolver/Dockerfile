FROM alpine:3.23

RUN apk add wget ca-certificates openssl

RUN wget -O /tmp/cert-chain-resolver.tar.gz https://github.com/zakjan/cert-chain-resolver/releases/download/1.0.4/cert-chain-resolver_linux_amd64.tar.gz \
    && tar -xzf /tmp/cert-chain-resolver.tar.gz -C /tmp \
    && mv /tmp/cert-chain-resolver_linux_amd64/cert-chain-resolver /usr/local/bin/cert-chain-resolver \
    && rm -rf /tmp/cert-chain-resolver_linux_amd64

# Task metadata
LABEL io.oplet.task.meta.name="Certificate chain resolver"
LABEL io.oplet.task.meta.description="A tool to generate a bundled certificate chain"
LABEL io.oplet.task.meta.author="Bornholm"
LABEL io.oplet.task.meta.url="https://github.com/bornholm/oplet/misc/tasks/cert-chain-resolver"

# Environment variable inputs
LABEL io.oplet.task.inputs.server_address.label="Server address"
LABEL io.oplet.task.inputs.server_address.type="text"
LABEL io.oplet.task.inputs.server_address.description="The server address in '<addr>:<port>' format"
LABEL io.oplet.task.inputs.server_address.required="true"

COPY --chmod=+x run.sh /usr/local/bin/run.sh

CMD ["/usr/local/bin/run.sh"]