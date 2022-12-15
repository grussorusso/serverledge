#!/bin/sh
bin/serverledge-cli create -f func --memory "$2" --src examples/hello.py --runtime python310 --handler "hello.handler" -H "$1"
bin/serverledge-cli create -f fib --memory "$2" --src examples/fibonacciNout.py --runtime python310 --handler "fibonacciNout.handler" -H "$1"