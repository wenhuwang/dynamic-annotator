FROM debian:stretch-slim

WORKDIR /

COPY dynamic-annotator /usr/local/bin

ENTRYPOINT ["dynamic-annotator"]