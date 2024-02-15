compile:
	GOOS=linux GOARCH=amd64 go build

docker: compile
	docker build .