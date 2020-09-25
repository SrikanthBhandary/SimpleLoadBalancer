FROM golang:alpine
RUN mkdir -p  /appservers
COPY ./appservers /appservers
WORKDIR /appservers
RUN go build appservers.go
CMD "./appservers"
