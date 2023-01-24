To test the offloading mechanism in a single (local) machine,
we can run two Serverledge nodes on the same host, allowing only one of
the nodes (i.e., the Cloud) to execute functions.

Run the following commands from the root directory of the repository:

1. Start Etcd

	bash scripts/start-etcd.sh

2. Start the "Edge" node

	bin/serverledge examples/local_offloading/confEdge.yaml

3. Start the "Cloud" node

	bin/serverledge examples/local_offloading/confCloud.yaml

4. Create and invoke a function

	bin/serverledge-cli create -f func --memory 256 --src examples/isprime.py --runtime python310 --handler "isprime.handler" 
	bin/serverledge-cli invoke -f func -p "n:17"

