FROM golang:latest

# create workdir
ADD . /app/
WORKDIR /app

# copy all file from to workdir
COPY . .

# instal psql
RUN apt-get update
RUN apt-get -y install postgresql-client

# build go app
RUN go mod download
RUN go build -o main cmd/gophermart/main.go
CMD ["./main"]
