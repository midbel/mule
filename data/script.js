console.log("foobar", 100, true, [1, 2])
console.log({name: "foobar", age: 100})

let obj = JSON.parse('{"name": "foobar", "age": 100}')
JSON.stringify(obj)

function fib(n, x = 10) {
	return n + 1
}