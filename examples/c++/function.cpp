#include "rapidjson/document.h" 
#include "rapidjson/filewritestream.h" 
#include "rapidjson/writer.h" 
#include <fstream> 
#include <iostream> 
#include <cstdlib>


  
using namespace std; 
using namespace rapidjson; 

// This is the function code
void fun (Document& params, Document& results) {
    if (!params.HasMember("a") || !params.HasMember("b"))
	    return;
    if (!params["a"].IsInt() || !params["b"].IsInt())
	    return;
    int a = params["a"].GetInt();
    int b = params["b"].GetInt();

    // Add data to the JSON document with results
    results.AddMember("Sum", a+b, results.GetAllocator()); 
}

  
int main() 
{ 
    // Open the input file 
    ifstream file(std::getenv("PARAMS_FILE")); 
    // Read the entire file into a string 
    string json((istreambuf_iterator<char>(file)), 
                istreambuf_iterator<char>()); 
  
    // Create a Document object  to hold the JSON data 
    Document params; 
  
    // Parse the JSON data 
    params.Parse(json.c_str()); 
  
    // Check for parse errors 
    if (params.HasParseError()) { 
        cerr << "Error parsing JSON: " << params.GetParseError() << endl; 
        return 1; 
    } 
  
  
    Document d; 
    d.SetObject(); 
  
    fun(params, d);

    StringBuffer buffer;
    Writer<StringBuffer> writer(buffer);
    d.Accept(writer);

    // Open the output file 
    std::ofstream outfile(std::getenv("RESULT_FILE")); 
    // Output {"project":"rapidjson","stars":11}
    outfile << buffer.GetString() << std::endl;

    return 0; 
}

