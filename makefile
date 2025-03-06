.PHONY: all clean

all: Podcasts

Podcasts: *.go
	go build -o Podcasts

clean:
	rm -f Podcasts
	go clean
