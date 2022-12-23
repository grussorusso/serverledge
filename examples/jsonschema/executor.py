from http.server import BaseHTTPRequestHandler, HTTPServer
import time
import os
import sys
import importlib
import json
import jsonschema
import function

hostName = "0.0.0.0"
serverPort = 8080


class Executor(BaseHTTPRequestHandler):
    def do_POST(self):
        content_length = int(self.headers['Content-Length']) 
        post_data = self.rfile.read(content_length) 
        request = json.loads(post_data.decode('utf-8'))

        if not "invoke" in self.path:
            self.send_response(404)
            self.end_headers()
            return

        try:
            params = request["Params"]
        except:
            params = {}

        if "context" in os.environ:
            context = json.loads(os.environ["CONTEXT"]) 
        else:
            context = {}


        response = {}

        try:
            result = function.handler(params, context)
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
    srv = HTTPServer((hostName, serverPort), Executor)
    try:
        srv.serve_forever()
    except KeyboardInterrupt:
        pass
    srv.server_close()

