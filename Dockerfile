FROM golang:stretch as build
RUN mkdir /go/src/SmartAndClassy
WORKDIR /go/src/SmartAndClassy
COPY . /go/src/SmartAndClassy
RUN go get .
RUN go build

FROM debian:bookworm
RUN mkdir /app
COPY --from=build /go/src/SmartAndClassy/SmartAndClassy /app/classyfy
WORKDIR /app
ENTRYPOINT classyfy
