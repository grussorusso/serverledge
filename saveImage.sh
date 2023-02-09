podman build -t grussorusso/serverledge-python310 -f images/python310/Dockerfile .
rm image.tar
podman save --output image.tar grussorusso/serverledge-python310
