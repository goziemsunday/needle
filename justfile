build:
    go build -o bin/needle .

run *args:
    go run . {{args}}

test:
    go test ./...

clean:
    rm -rf bin/

get:
    go get ./...