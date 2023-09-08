function handler(params, context) {
    console.log("params: ", params);
    console.log("params[input]: ", params["input"]);
    return parseInt(params["input"], 10) +5;
}

module.exports = handler;