all:
	-rm $(CMD)
	GOOS=linux GOARCH=amd64 go build -a
	docker build -t demo .
