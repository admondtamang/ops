FROM golang:1.26-alpine AS builder
  WORKDIR /app
  COPY . .
  RUN go build -o server .
                                                                                                                                                                         
  FROM alpine:3.19
  COPY --from=builder /app/server /server                                                                                                                                
  EXPOSE 8080                            
  CMD ["/server"]