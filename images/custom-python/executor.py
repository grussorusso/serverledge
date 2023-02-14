# Python3 server
from http.server import BaseHTTPRequestHandler, HTTPServer
from multiprocessing import Process
import requests
import time
import json
import sys
import os
from sklearn.feature_selection import mutual_info_classif, SelectKBest
from sklearn.model_selection import train_test_split
from sklearn.preprocessing import LabelEncoder
import pandas as pd
import numpy as np


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
        print("Function invocation process started.")
        content_length = int(self.headers['Content-Length']) 
        post_data = self.rfile.read(content_length) 
        request = json.loads(post_data.decode('utf-8'))
        id = request["Id"]
        response = {}
        try:

            '''
            Questo esempio prende un dataset da 60k istanze e 20 attributi; su esso esegue una 
            feature selection basata sulla mutual information. Tale meccanismo permette di selezionare 
            gli attribui migliori stimando la dipendenza tra due variabili. In questo modo è 
            possibile scartare gli attributi non utili ai fini dell'apprendimento, tenendo solo quelli
            che contengono una maggior quantità di informazione. 
            '''

            df = pd.read_csv("https://raw.githubusercontent.com/msalvati1997/mushrooms_classificator/main/secondary_data.csv")
    
            # Convert nominal values into real ones
            df['class'] = df['class'].replace('p',1)
            df['class'] = df['class'].replace('e',0)
            labelencoder=LabelEncoder()
            for column in df.columns:
                if column!= 'class' and column!='stem-height' and column!='stem-width' and column!='cap-diameter':
                    df[column] = labelencoder.fit_transform(df[column])

            # Split it into training and testing set
            X = df.drop(['class'], axis=1)
            Y=df['class']
            y = np.array(Y, dtype = 'float32')
            x = np.array(X, dtype = 'float32')
            x_train, x_test, y_train,y_test = train_test_split(x,y,train_size=0.9, random_state=50)

            # Train the model 
            model = SelectKBest(mutual_info_classif)
            model.fit(x_train, y_train)
            response["Result"] = "OK"
            response["Success"] = True
            response["Id"] = id

        except Exception as e:
            print(e, file=sys.stderr)
            response["Success"] = False
        # Try to send the response. It could fail if a migration occurs
        try: 
            self.send_response(200)
            self.send_header("Content-type", "application/json")
            self.end_headers()
            self.wfile.write(bytes(json.dumps(response), "utf-8"))
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
        print("Function invocation process ended.")

class FallbackListener(BaseHTTPRequestHandler):
    def do_POST(self):
        print("Fallback listener has been called.")
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

def start_listening(webServer):
    webServer.serve_forever()

if __name__ == "__main__":
    sys.stdout = Unbuffered(sys.stdout) # Use unbuffered output

    executorServer = HTTPServer((hostName, serverPort), Executor)
    fallbackListener = HTTPServer((hostName, serverPort+1), FallbackListener)
    
    # Start the listeners on different ports, using different processes
    Process(target=start_listening, args=[executorServer]).start()
    Process(target=start_listening, args=[fallbackListener]).start()
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

