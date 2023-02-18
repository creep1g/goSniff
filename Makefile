clean:
	sudo rm /bin/goSniff

build: 
	go build -o bin/goSniff

install:
	sudo go build -o /bin/goSniff


