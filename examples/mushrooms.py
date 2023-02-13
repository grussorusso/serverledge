from sklearn.feature_selection import mutual_info_classif
from sklearn.model_selection import train_test_split
from sklearn.feature_selection import SelectKBest
from sklearn.preprocessing import LabelEncoder
import pandas as pd
import numpy as np

'''
Questo esempio prende un dataset da 60k istanze e 20 attributi; su esso esegue una 
feature selection basata sulla mutual information. Tale meccanismo permette di selezionare 
gli attribui migliori stimando la dipendenza tra due variabili. In questo modo è 
possibile scartare gli attributi non utili ai fini dell'apprendimento, tenendo solo quelli
che contengono una maggior quantità di informazione. 
'''

def handler(params, context):
    return train_model


def train_model():
    
    # Acquire the dataset
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

    return "OK"
    