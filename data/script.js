console.log("foobar", 100, true, [1, 2])
console.log({name: "foobar", age: 100})

let obj = JSON.parse('{"name": "foobar", "age": 100}')
JSON.stringify(obj)

function fib(n, x = 10) {
	console.log("n = ", n)
	console.log("x = ", x)
	return n + 1
}

fib(100, 10)

obj = {
	name: "foobar",
	age: 100,
	test: function() {
		console.log(this)
		return "testmeifyoucan"
	},
	test2() {
		console.log("test2")
	}
}

console.log('>> before: ', obj.name)
obj.name = "foo"
obj.age = 5
console.log('<< after1: ', obj.name, obj.age)
obj["name"] = "bar"
console.log('<< after2: ', obj.name, obj.age)

const arr1 = [1, 2]
arr1[0] = "foobar"
console.log(arr1)


console.log(obj.test())
console.log(obj.test2())

let v = 2
let r = (v = v+2 , 3+3) * 2
console.log("v(4) = ", v)
console.log("r(12) = ", r)

const arr = [1, 2, 3]
console.log("array length", arr.length)
console.log("pi", Math.PI)

const arrow1 = x => x
const arrow2 = (x, y) => x + y
const arrow3 = (x, y) => {
       return [x + y]
}

switch (expr) {
case "first":
	1+1
	console.log("test first")
	break;
case "second":
	2+1
	console.log("test second")
case "third":
	break;
default:
	console.log("fallback")
}