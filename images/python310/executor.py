# Python 3 server example
from http.server import BaseHTTPRequestHandler, HTTPServer
from multiprocessing import Process
import requests
import time
import os
import sys
import importlib
import json

hostName = "0.0.0.0"
serverPort = 8080

fallbackAddressesFile = "/tmp/_executor_fallback_addresses.txt"
executed_modules = {}
added_dirs = {}

class Unbuffered(object):
    '''
    'Unbuffered mode' allows for log messages to be immediately dumped to the stream instead of being buffered.
    '''
    def __init__(self, stream):
        self.stream = stream
    def write(self, data):
        self.stream.write(data)
        self.stream.flush()
    def writelines(self, datas):
        self.stream.writelines(datas)
        self.stream.flush()
    def __getattr__(self, attr):
        return getattr(self.stream, attr)

class Executor(BaseHTTPRequestHandler):
    def do_POST(self):
        print("INVOKE CHIAMATO")
        content_length = int(self.headers['Content-Length']) 
        post_data = self.rfile.read(content_length) 
        request = json.loads(post_data.decode('utf-8'))

        if not "invoke" in self.path:
            self.send_response(404)
            self.end_headers()
            return

        handler = request["Handler"] 
        handler_dir = request["HandlerDir"]

        try:
            params = request["Params"]
        except:
            params = {}

        if "context" in os.environ:
            context = json.loads(os.environ["CONTEXT"]) 
        else:
            context = {}

        if not handler_dir in added_dirs:
            sys.path.insert(1, handler_dir)
            added_dirs[handler_dir] = True

        # Get module name
        module,func_name = os.path.splitext(handler)
        func_name = func_name[1:] # strip initial dot

        response = {}

        try:
            # Import module
            if not module in executed_modules:
                exec(f"import {module}")
                executed_modules[module] = True

            # Call function
            mod = importlib.import_module(module)
            result = getattr(mod, func_name)(params, context)
            response["Result"] = json.dumps(result)
            response["Success"] = True
        except Exception as e:
            print(e, file=sys.stderr)
            response["Success"] = False
        # Try to send the response. It could fail if a migration occurs
        try: 
            self.send_response(200)
            self.send_header("Content-type", "application/json")
            self.end_headers()
            self.wfile.write(bytes(json.dumps(response), "utf-8"))
            print("INVOKE FINITO")
        except ConnectionResetError:
            print("Seems like this container has been migrated. The occurred exception is ",sys.exc_info()[0])
            self.close_connection
            
            # Acquire the possible new node addresses from the local file containing fallbackNode IPs
            with open(fallbackAddressesFile, 'r') as f:
                fallbackNodes = []
                for line in f.readlines():
                    fallbackNodes.append(line.rstrip())
                print("Fallback nodes acquired:")
                print(fallbackNodes)

            # Send the result to the new node    
            payload = json.dumps(response)
            for node in fallbackNodes:
                try:
                    print("Trying to contact http://"+ str(node) +":1323/receiveResultAfterMigration")
                    requests.post('http://'+ str(node) +':1323/receiveResultAfterMigration', json = payload)
                except:
                    print("Failed to send the result to the node: " + str(node))
                    pass
        print("Response sent.")

class FallbackListener(BaseHTTPRequestHandler):
    def do_POST(self):
        print("FALLBACK CHIAMATO")
        content_length = int(self.headers['Content-Length']) 
        post_data = self.rfile.read(content_length) 
        request = json.loads(post_data.decode('utf-8'))
        fallbackAddresses = request["FallbackAddresses"]

        print("Fallback nodes acquired:")
        print(fallbackAddresses)
        
        #Write the address to a local file
        with open(fallbackAddressesFile, 'w') as f:
            for ip in fallbackAddresses:
                f.write(ip+'\n')
            print("Fallback node addresses stored.")
        
        response = {}
        response["Success"] = True

        self.send_response(200)
        self.send_header("Content-type", "application/json")
        self.end_headers()
        self.wfile.write(bytes(json.dumps(response), "utf-8"))
        print("FALLBACK FINITO")

def turn_on(webServer):
    webServer.serve_forever()

if __name__ == "__main__":
    sys.stdout = Unbuffered(sys.stdout) # Use unbuffered output

    

    executorServer = HTTPServer((hostName, serverPort), Executor)
    fallbackListener = HTTPServer((hostName, serverPort+1), FallbackListener)
    
    # Start the listeners on different ports, using different processes
    Process(target=turn_on, args=[executorServer]).start()
    Process(target=turn_on, args=[fallbackListener]).start()
    print("Container services correctly initialized.")

    try:
        while True:
            time.sleep(60)    
    except KeyboardInterrupt:
        executorServer.server_close()
        fallbackListener.server_close()
        pass
    
    print("\nServices closed. The container will close.\n")
    os._exit(1)

