# Language Experiment

This is a playground for PL ideas. My goal is to have a safe elegant language 
while maintaining simplicity. This is clearly subjective.

- Sum types.
- Static typing.
- Concurrent safety.
- ...

## First-class support for deploying

What does this mean?

- Builtin support for bundling resources, hot reload during development.
- Automatic Docker builds?

## Concurrency options

The appeal of Go's concurrency model is that any normal synchronous function
can be called asynchronously. The downside of its model is that there are no 
guarantees around concurrent access - each type must manage its own synchronisation
manually. How do we solve this?

### Immutable values?

All values are immutable. However to improve ergonomics any mutating operations will 
automatically perform a copy. In reality the implementation should avoid copies 
wherever possible while maintaining the semantics of immutability.

Each parameter has two potential taints:

1. A mutation taint that states whether the function mutates the 
   parameter value.
2. An asynchronous taint that states whether the parameter is passed to an
   asynchronous construct.
   
Taints are transitive.

```
// This function does not mutate "s", does not pass "s" to any functions that do,
// and does not create any threads referencing "s". This guarantees that dump() is
// pure and synchronous.
fn dump(s Stack<string>) {
    for v in s {
        println(v)
    }
}

let a = new Stack<string>()
a.push("a")     // Mutate.
dump(a)         // Reference (synchronous+pure).
go dump(a)      // Copy (asynchronous).
a.x = 1.11      // Mutate.
```

### Automatic ownership management

Like Rust, but without the boilerplate. Similar to [D](https://dlang.org/blog/2019/07/15/ownership-and-borrowing-in-d/).

Rules:

1. Any value passed to an asynchronous construct will have its ownership transferred.
2. The `copy` operator creates a clone of a value.
3. Sub-values of compound types (fields, array values, etc.) cannot be transferred,
   only standalone values.
   
### [Active Object](https://en.wikipedia.org/wiki/Active_object)

Concept: concurrent-safe facades for synchronous objects are automatically
generated by the compiler.

eg. Given the following unsynchronised class:

```
enum Scalar = float|string|bool

enum Value {
case Scalar(Scalar)
case List([Scalar])
case Hash({string: Scalar})
}

class Redis {
    let values: {string: Value}

    fn len(key: string): int? {
        let value = values[key] else return
        switch value {
        case .List(list):  
            return list.size()

        case .Hash(hash):
            return hash.size()

        case .Scalar(scalar):
            if let str = scalar as .string {
                return str.size()
            }
        }
    }

    // Append one or more elements to a list.
    fn rpush(key: string, scalar: Scalar): error?
        if let value = values.setDefault(key, .List([])) {
            list.append(scalar)
        } else {
            return error("expected a list at {key}")
        }
    }

    // Length of a list.
    fn llen(key: string): int? {
        if let value = values[key]; list = value as .List {
            return list.size()
        }
    }

    fn lindex(key: string, index: int): Scalar? {
        if let value = values[key]; list = value as .List {
            return list[index]
        }
    }

    // Set a field of a hash value.
    fn hset(key: string, field: string, value: Scalar): error? {
        if let value = values.setDefault(key, .Hash({})) {
            hash[field] = value
        } else {
            return error("expected a hash at {key}")
        }
    }

    fn hget(key: string, field: string): Scalar? {
        if let value = values[key]; hash = value as .Hash {
            return hash[field]
        }
    }

    fn hlen(key: string): int? {
        if let value = values[key]; hash = value as .Hash {
            return hash.size()
        }
    }

    fn del(key: string) {
        values.delete(key)
    }

    fn exists(key: string): bool {
        return values.contains(key)
    }
}
```

### Actors?

- Actor methods cannot return values.
- Invoking any method (including the constructor) on an actor enqueues the call on its mailbox.
- All messages to the actor are applied synchronously.
- Actors cannot contain public fields.
- Any value passed into an actor will transfer ownership to the actor.
- If an actor named "main" exists it will be the main entry point rather than the "main" function.

#### Actor functions?

```
actor fn poll(msg: string) {
   print(msg)
}

let p = start poll
p("hello")
```

Bit ugly?

#### Normal actors

```
actor Owner {
    let pet: Pet?

    init(pet: Pet?) {
        self.adopt(pet)
    }

    fn adopt(pet: Pet) {
        self.pet = pet
        feed("seed")
    }

    fn feed(food: string) {
        if let pet = pet {
            pet.feed(food)
        }
    }
}

actor main {
    init() {
        let pet = Pet("Lucius")
        //let pets = [pet]
        // Start Actor.
        //let owner = Owner(pets[0])   // Error: can't transfer ownership of an element.
        let owner = start Owner(pet)   // Create and start the actor.
        //pet.feed("moo")              // Error: pet is owned by "owner"

        // Send some messages.
        owner.adopt(pet)
        owner.feed("seed")

        kill(owner)
    }
}
```

This compiles to the following Go code.

```
func Kill(actor Actor, err error) {
    actor.Supervisor().Kill(actor, err)
}

type Supervisor interface {
    Kill(actor Actor, err error)
}

type Actor interface {
    // Get the Supervisor for the Actor.
    Supervisor() Supervisor
    // Triggered when the Actor is being killed. Should perform cleanup.
    OnKill()
}

type ActorImpl struct {
    mailbox    chan func()
    supervisor Supervisor
}

type Owner struct {
    // Actor state.
    pet *Pet
}


func NewOwner(supervisor Supervisor, pet *Pet) *Owner {
    o := &Owner{
        mailbox: make(chan func(), 100),
        supervisor: supervisor,
    }
    go o.run()
    o.send(func() { o.fnInit(pet) })
    return o
}

func (o *Owner) Supervisor() Supervisor {
    return o.supervisor
}

func (o *Owner) OnKill() {
    close(o.mailbox)
}

// Actor main loop.
func (o *Owner) run() {
    for msg := range o.mailbox {
        msg()
    }
}

func (o *Owner) send(f func()) {
    select {
    case o.mailbox <- f:
    default:
        actors.Kill(o, errors.New("mailbox full"))
    }
}

func (o *Owner) SendAdopt(pet *Pet) { o.send(func() { o.fnAdopt(pet) }) }

func (o *Owner) SendFeed(food string) { o.send(func() { o.fnFeed(food) }) }

func (o *Owner) SendStop() { o.send(func() { o.fnStop() }) }

func (o *Owner) fnInit(pet *Pet) {
    o.fnAdopt(pet)
}

func (o *Owner) fnAdopt(pet *Pet) {
    o.pet = pet
    o.fnFeed("seed")
}

func (o *Owner) fnFeed(food string) {
    if pet := o.pet; pet != nil {
        pet.Feed(food)
    }
}

type Main struct {
    ActorImpl
}

func main() {
    pet := NewPet()
    owner := NewOwner(pet)
    owner.SendAdopt(pet)
    owner.SendFeed("seed")
    actors.Kill(owner, nil)
}
```

[Akka example](https://alvinalexander.com/scala/how-to-communicate-send-messages-scala-akka-actors):

```scala
import akka.actor._

case object PingMessage
case object PongMessage
case object StartMessage
case object StopMessage

class Ping(pong: ActorRef) extends Actor {
    var count = 0
    def incrementAndPrint { count += 1; println("ping") }
    def receive = {
        case StartMessage =>
            incrementAndPrint
            pong ! PingMessage
        case PongMessage =>
            incrementAndPrint
            if (count > 99) {
                sender ! StopMessage
                println("ping stopped")
                context.stop(self)
            } else {
                sender ! PingMessage
            }
        case _ => println("Ping got something unexpected.")
    }
}

class Pong extends Actor {
    def receive = {
      case PingMessage =>
          println(" pong")
          sender ! PongMessage
      case StopMessage =>
          println("pong stopped")
          context.stop(self)
      case _ => println("Pong got something unexpected.")
    }
}

object PingPongTest extends App {
    val system = ActorSystem("PingPongSystem")
    val pong = system.actorOf(Props[Pong], name = "pong")
    val ping = system.actorOf(Props(new Ping(pong)), name = "ping")
    // start the action
    ping ! StartMessage
    // commented-out so you can see all the output
    //system.shutdown
}
```

In langx:

```
// Actor traits may only be applied to actors.
actor trait Pinger {
  fn ping(ping: Pinger)
}

actor Ping: Pinger {
    let count = 0

    pub fn start(pong: Pong) {
        incrementAndPrint()
        pong.ping(self)
    }

    pub fn ping(ping: Pinger) {
        incrementAndPrint()
        if count > 99 {
            println("ping stop")
            ping.stop()
            kill(self)
        } else {
            ping.ping(self)
        }
    }

    fn incrementAndPrint() {
        count++
        println("ping")
    }
}

actor Pong: Pinger {
    pub fn ping(ping: Pinger) {
        println("pong")
        ping.ping(self)
    }

    pub fn stop() {
        println("pong stop")
        kill(self)
    }
}

fn main() {
    let ping = Ping()
    let pong = Pong()
    ping.start(pong)
}

```

## Classes

```
pub class Vector: Stringer {
    // All fields are given default values. One difference from Go is that
    // arrays, maps, and classes are given default-constructor values.
    let x, y, z : float32

    // A default constructor is always provided for all public fields.
    // In this case it would be equivalent to:
    //
    //     constructor(x:float = 0, y:float = 0, z:float = 0)

    pub fn length():float { // Pure.
        return Math.sqrt(x * x + y * y + z * z)
    }

    pub fn add(other:Vector) { // Impure.
        x += other.x
        y += other.y
        z += other.z
    }

    // "override" can be specified when implementing traits to ensure that changes to the
    // interface don't result in methods beign orphaned.
    override pub fn string(): string {
        return "Vector({x}, {y}, {z})"
    }
}
```

## Generics

```
class Stack<T>: Iterable<T> {
    let stack: [T]   // Backed by an array.

    pub fn push(v: T) {
        stack.append(v)
    }

    // Implements Iterable<T>
    override pub fn iterator(): Iterator<T> {
        return stack.iterator()
    }
}
```

Generic functions:

```
fn map<T, U>(l: [T], f fn(v T):U): [U] {
    let out: [U]
    for v in l {
        out.append(f(v))
    }
    return out
}

let ints = [1, 2, 3]
let floats = map(ints, fn(v int) float {
    return float(v)
})
```

## Arrays

```
let a: [string]         // Explicitly typed.
let a = ["hello"]       // Type inference.
```

## Maps

```
let a: {string: Vector}              // Explicitly typed.
let b = {"hello": Vector(x:1, y:2, z:3)}   // Type inference.
```

## Sets

```
let a: {string}         // Explicitly typed.
let a = {"hello"}        // Type inference.
```

## Type aliases?

Creates an alias for an existing type, with its own set of methods etc.


```
alias Number float {
    // TODO: Constraints?
    constraint self >= 1 && self <= 10

    fn midpoint() float {
        return this / 2.0
    }
}

```

## Channels?

Channels should be used to pass values between threads.

```
let a = new chan<Vector32>()

let v = Vector32{1, 2, 3} 
v.x = 2   // Mutate
a <- v    // Copy
```

## Interfaces


Structural typing and traditional interfaces are complementary in that the former is
more consumer-centric, while the latter is more provider-centric. To that end, langx 
interfaces support both.

Interfaces are fairly straightforward:

```
interface Pet {
    // Method with a default implementation. Can still be overridden.
    fn description(): string {
        return "{name} is {age} years old"
    }

    // Methods without implementations must be provided.
    fn mood(): string

    // Fields without a default value must be provided by implementations.
    let name: string
    let age: int
}
```


Structural typing usage:

```
class Dog {
    pub fn mood(): string {
        return "happy
    }

    pub let name: string
    pub let age: int
}

let dog: Pet = Dog(name: "Fido", age: 8)
```

```
class Dog: Pet {
    pub fn mood(): string {
        return "happy
    }

    pub let name: string
    pub let age: int
}
```

## Constructors

Constructor arguments are *always* named? Positional arguments are not supported?

```
class Vector {
    let x, y, z: float

    // Static factory method.
    static fn unit(): Vector {
        return Vector(y: 1)
    }
}

fn f() {
    // A default constructor is generated if not otherwise provided.
    let a = Vector(x: 1)
}
```

## Sum types / enums

Very similar to Swift.

```
enum Result<T> {
    case Value(T)
    case Error(error)
}

enum Optional<T> {
    case None
    case Value<T>
}

let result = Result.Value("hello world")

switch (result) {
case .Value(value):
case .Error(err):
}
```

The default value for an enum is the first case, only if it is untyped. If all cases
are typed (eg. `Result<T>` above) then there is no possible default value.

Go code:

```go
type ResultString interface { resultStringCase() }

type ResultStringValue string
func (ResultStringValue) resultStringCase() {}

type ResultStringError error
func (ResultStringError) resultStringCase() {}

var result = ResultString(ResultStringValue("hello world"))

switch result := result.(type) {
case ResultStringValue:
    value := result

case ResultStringError:
    err := result
}
```

Support for anonymously combining types into enums:

```
enum Option<T> {
    case Value(T)
    case None
}

// This will merge the Option<T> with error to create a single enum:
//
// enum Anonymous {
//   case Value(string)
//   case None
//   case error
// }
//
// Do we want this vs. the Option becoming a first class case?
//
// Upside is you can return any literal that can be inferred, downside is you can't 
// convert to an Option.
fn f(): Option<string>|error {
    return "hello"
}
```


## Pattern matching

```
let tuples = [("a", 123), ("b", 234)]

for tuple in tuples {
    match tuple {
    case ("a", n):
        println("a #{n}")

    default:
        println(tuple[0], tuple[1])
    }
}
```

```go
type StringIntTuple struct {
    A string
    B int
}
var tuples = []StringIntTuple{
    {A: "a", B: 123},
    {A: "b", B: 234},
}

for _, tuple := range tuples {
    switch {
    case tuple.A == "a":
        n := tuple.B
        fmt.Sprintf("a %d", n)

    default:
        fmt.Println(tuple.A, tuple.B)
    }
}
```

## For loop

```
for value in array {
}

for (index, value) in array {
}

for key in map {
}

for (key, value) in map {
}

for value in set {
}

for value in channel {
}
```

## Error handling?

```
enum MyError:error {
    case IOError(io.Error)
    case UserError(string)
}

fn sub():string throws {
    return ""
}

fn function():string throws {
    if false {
        throw MyError.UserError("something is false")
    }
    let a = try sub() // Rethrow
    if let a = try sub() {
    }
    return "hello"
}
```

## Native optional type

```
// Builtin type definition.
enum Optional<T> {
    case None
    case Some<T>
}

let a = Optional.Some("hello world")

fn f() {
    // "if let" is shorthand for:
    //
    //      switch a {
    //      case .Some(b):
    //      default:
    //      }
    if let b = a {
    } else {
    }

    a = none
    // Shorthand for: a = Optional<string>.None
    a = "goodbye"
    // Shorthand for: a = Optional<string>.Some("goodbye")
}
```

## Imports

Like Go? Automatic imports?

Python style?

```
import <package>[ as <alias>][, ...]
from <package> import <symbol>[as <alias>][, ...]
```

eg.

```
import "github.com/alecthomas/participle" as parser
from "github.com/alecthomas/participle" import Parser
```

Or Go style only?

```
import <package>[ as <alias>][, ...]
```

eg.

```
import "github.com/alecthomas/participle" as parser
```

## Annotations?

Accessible via reflection. Mmmmmmmmmmm. Avoid for now, though there needs to be
some solution for eg. JSON encoding. Writing manual encoders/decoders sucks.

```
import "types"
from "types" import annotation

// Lets the compiler know that this class is intended to be an annotation.
@annotation(restrict=[types.Class, types.Field, types.Method])
class json {
    let omit:bool = false
}

class User {
    let name:string

    let email:string

    let age:int

    @json(omit=true)
    let ssn:string
}
```

## Compile time reflection

Ala [Zig](https://ziglang.org/#Compile-time-reflection-and-compile-time-code-execution). This is great.
I haven't given this much though, so I'm not sure what it would entail. An interpreter maybe?

## Interoperability with Go/C?

If the language is hosted by the Go runtime, should it support interoperability with Go? Or C?

Pros:
- large set of existing libraries

Cons:
- how does immutability interoperate with Go?
- limits the language to constructs supported by the Go runtime


## Resource lifetimes

For objects implementing `io.Closer`, `with` will close them at the end of the block.

```
with try f = os.open("/etc/passwd") {
}
```

As with `if try`, `with` blocks can have `catch` and `rethrow` alternates.

## Error handling

Errors are reported via enums:

```
fn open(path: string): File|error {
    return error("{path} not found")
}
```

How do we handle these elegantly? 

### Rust-style re-throw operator `?`?

```
let f = os.Open("/etc/passwd")?
```

Very convenient, but too magical?

### Error-specific `try` syntax in `if` and `with` blocks?

```
if|with try [<var> = ] <expr> <block>
catch [[<var>:]<type>] <block>
rethrow
```

```
with try file = os.open("/etc/passwd") {
    let scanner = new bufio.Scanner(file)
    for scanner.scan() {
        println(scanner.text())
    }
    return scanner.err()
} catch os.ErrNotExist {
    return
} rethrow

with try file = os.open("/etc/passwd") {
}

if try os.stat("/etc/passwd") {
} catch {
}
```

```rust
use std::io;
use std::fs;

fn read_username_from_file() -> Result<String, io::Error> {
    fs::read_to_string("hello.txt")
}
```

```
import ioutil

fn readUsernameFromFile(): bytes|error {
    return ioutil.readFile("hello.txt")
}

// <type>? is a shortcut for <type>|none
fn username(): string? {
    return "bob"
}
```

First example becomes:

```go
file, err := os.Open("/etc/passwd")
if err == nil {
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        fmt.Println(scanner.Text())
    }
    file.Close()
    return scanner.Err()
} else if err == os.ErrNotExist {
    return nil
} else {
    return err
}
```

## Examples

### Enums as Go

```
class Cat {
    let name : string
}

class Dog {
    let name : string
}

enum Animal {
    case Cat(Cat)
    case Dog(Dog)
    case A(int)
    case B(int)
    case Unkown
}

let animal : Animal = new Dog(name:"Fido")

if let dog = animal as Dog {
    animal = new Cat(name:"Kitty")
}

switch animal {
case Cat(cat):
case Dog(dog):
case A(n):
case B(n):
case Unknown:
}
```

Becomes:

```go
package main

import "fmt"

type Cat struct{ Name string }
type Dog struct{ Name string }

type Animal interface{ animal() }

type AnimalCat Cat
func (*AnimalCat) animal() {}

type AnimalDog Dog
func (*AnimalDog) animal() {}

type AnimalA int
func (AnimalA) animal() {}

type AnimalB int
func (AnimalB) animal() {}

type AnimalUnknown struct{}
func (*AnimalUnknown) animal() {}


func main() {
    var animal Animal = (AnimalB)(23)

	if __animal, ok := animal.(*AnimalDog); ok {
		dog := (*Dog)(__animal)
		fmt.Printf("%T\n", dog)
        
        animal = (*AnimalCat)(&Cat{Name: "Kitty"})

	}

	switch __animal := animal.(type) {
	case *AnimalCat:
		cat := (*Cat)(__animal)
		fmt.Printf("%T\n", cat)

	case *AnimalDog:
		dog := (*Dog)(__animal)
		fmt.Printf("%T\n", dog)

	case AnimalA:
		n := int(__animal)
		fmt.Printf("A %T\n", n)

	case AnimalB:
		n := int(__animal)
		fmt.Printf("B %T\n", n)

	case *AnimalUnknown:
	}
}
```

### Templated Enum

```
enum Option<T> {
    case Some(T)
    case None
}

let a : Option<string> = "hello"
let b : Option<String> = none
```

Becomes:

```go
type OptionString interface { optionString() }

type OptionStringSome string
func (OptionStringSome) optionString() {}

type OptionStringNone struct{}
func (OptionStringNone) optionString() {}

var a OptionString = OptionStringSome("hello")
var b OptionString = OptionStringNone{}
```
 
### Entity Component System

```
class Vector {
    let x, y, z : float
}

class Base {
    let position, direction : Vector
    let opacity : float
}

class Script {
    let source : string
}

enum Component {
    case Base(Base)
    case Script(Script)

    let slot() : int {
        switch self {
        case Base(_): return 0 
        case Script(_): return 1
        }
    }
}

class ECS {
    let free : [int]
    let entities : [[Component?]]

    // Create a new Entity.
    fn create() : int {
        if (free.size() > 0) {
            return free.pop()
        }
        let id = entities.size()
        entities.append([])
        return id
    }

    fn delete(id : int) {
        free.push(id)
        entities[id] = []
    }

    fn assign(id : int, component : Component) {
        let components = entities[id]
        let slot = component.slot()

        if (slot >= components.size()) {
            components.resize(slot + 1)
        }
        components[slot] = component
    }

    fn unassign(id : int, component : Component) {
        entities[id][component.slot()] = none
    }
}
```
