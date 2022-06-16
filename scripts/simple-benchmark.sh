#/bin/bash


bin/serverledge-cli delete -f sieve
set -e

bin/serverledge-cli create -f sieve --memory 128 --src examples/sieve.js --runtime nodejs17ng --handler "sieve.js" && sleep 1

echo '{"Params":{},"QoSClass":0}' > /tmp/json
ab -l -p /tmp/json -T application/json -c 10 -n 10000 http://127.0.0.1:1323/invoke/sieve | tee ab_output.txt
