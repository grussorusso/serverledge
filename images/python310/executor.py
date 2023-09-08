# Python 3 server example
from http.server import BaseHTTPRequestHandler, HTTPServer
import time
import os
import sys
import importlib
import json

hostName = "0.0.0.0"
serverPort = 8080

executed_modules = {}
added_dirs = {}

class Executor(BaseHTTPRequestHandler):
    def do_POST(self):
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

        self.send_response(200)
        self.send_header("Content-type", "application/json")
        self.end_headers()
        self.wfile.write(bytes(json.dumps(response), "utf-8"))



if __name__ == "__main__":
    webServer = HTTPServer((hostName, serverPort), Executor)
    print("Server started http://%s:%s" % (hostName, serverPort))

    try:
        webServer.serve_forever()
    except KeyboardInterrupt:
        pass

    webServer.server_close()
    print("Server stopped.")

