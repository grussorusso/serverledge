#!/bin/sh
bin/serverledge-cli create -f func --memory "$2" --src examples/hello.py --runtime python310 --handler "hello.handler" -H "$1"
bin/serverledge-cli create -f fib --memory "$2" --src examples/fibonacci_nout.py --runtime python310 --handler "fibonacci.handler" -H "$1"