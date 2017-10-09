FROM golang:latest
WORKDIR /go/src/github.com/seralexeev/hlcup-go/
RUN go get -u github.com/mailru/easyjson
RUN go get -u github.com/valyala/fasthttp
COPY app .
# RUN go-wrapper download
# RUN go-wrapper install
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

FROM alpine:latest  
RUN apk add --update curl
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /go/src/github.com/seralexeev/hlcup-go/app .
COPY run.sh .
RUN ["chmod", "+x", "run.sh"]
CMD ["./run.sh"]