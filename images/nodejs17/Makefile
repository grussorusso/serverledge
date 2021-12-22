TAG=grussorusso/serverledge-nodejs17

all: build_image

build_image:
	cp ../../bin/executor .
	docker build -t $(TAG) .

push: build_image
	docker push $(TAG)

.PHONY: push build_image
