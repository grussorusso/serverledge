#!/bin/sh

THIS_DIR=$(dirname "$0")

"$THIS_DIR"/../../bin/serverledge-cli compose -fc sequence --memory 40 --src examples/inc.py --runtime python310 --handler "inc.handler"