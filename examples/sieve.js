// Example ported from TinyFaaS (https://github.com/OpenFogStack/tinyFaaS/blob/master/examples/sieve-of-erasthostenes/index.js)
//
module.exports = (params, ctx) => {
	var max;

	try {
		max = parseInt(params["n"],10);
	} catch (err) {
		max = 1000;
	}

	let sieve = [], i, j, primes = [];
	for (i = 2; i <= max; ++i) {

		if (!sieve[i]) {
			primes.push(i);
			for (j = i << 1; j <= max; j += i) {
				sieve[j] = true;
			}
		}
	}

	return primes
}
