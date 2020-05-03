FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /root/

EXPOSE 13734

# create and populate ~/bin with appropriate executable.
COPY ./instance/deployment/ ./

ENV PATH="./bin:$PATH"

CMD httpForwarder
