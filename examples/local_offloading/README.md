To test the offloading mechanism in a single (local) machine,
we can run two Serverledge nodes on the same host, allowing only one of
the nodes (i.e., the Cloud) to execute functions.

Run the following commands from the root directory of the repository:

1. Start Etcd

	bash scripts/start-etcd.sh

2. Start the "Edge" node

	bin/serverledge examples/local_offloading/conf1.yaml

3. Start the "Cloud" node

	bin/serverledge examples/local_offloading/conf2.yaml

4. Create and invoke a function

	bin/serverledge-cli create -f func --memory 600 --src examples/hello.py --runtime python310 --handler "hello.handler" 
	bin/serverledge-cli invoke -f func -p "a:2" -p "b:3"

