all: build_image

build_image:
	cp ../../bin/executor .
	docker build -t grussorusso/serverledge-python310 .

push: build_image
	docker push grussorusso/serverledge-python310

.PHONY: push build_image
