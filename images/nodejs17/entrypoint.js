let path = require('path');
let fs = require("fs");

const handler = process.env.HANDLER // e.g. "myfile.js"
const handler_dir = process.env.HANDLER_DIR
const result_file = process.env.RESULT_FILE

var params = {}
if (process.env.PARAMS !== "undefined") {
    params = process.env.PARAMS
}

var context = {}
if (process.env.CONTEXT !== "undefined") {
    context = process.env.CONTEXT
}

let h = require(path.join(handler_dir, handler))

result = h(params, context)
fs.writeFileSync(result_file, JSON.stringify(result) , 'utf-8');

