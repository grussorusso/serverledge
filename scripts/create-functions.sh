#!/bin/sh
bin/serverledge-cli create -f func --memory 600 --src examples/hello.py --runtime python310 --handler "hello.handler" -H "$1"
bin/serverledge-cli create -f fib --memory 600 --src examples/fibonacci.py --runtime python310 --handler "fibonacci.handler" -H "$1"