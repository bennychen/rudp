## About

This is a go/csharp/typescript transplant of a RUDP implementation, originally written in C by the lengendary developer cloudwu from China.

The original algorithm explaination is at [cloudwu's blog](http://blog.codingnow.com/2016/03/reliable_udp.html) (In Chinese)
and [here](https://github.com/cloudwu/rudp) is the source c code repo.

## APIs

The following functions are provided:

**Create**

creates a RUDP object

- sendDelay: after how long we should send messages

- expiredTime: after how long messages in history should be cleared

**Send**

sends a new message out

**Recv**

receives message and returns the size of the new message

if 0 returned, there is no new mesasge

if -1 returned, it's a corrupted connection

**Update**

should be called every frame with the time tick, or when a new package is coming.

- received: the actual UDP package we received

- sz: the size of the package

the package returned from this function should be sent out with UDP.

## How to Use

Caller only needs to call Send & Receive. `Send` doesn't actually send data out, it just piles data into RUDP object and generates packages when next send time is reached. Also, `Recv` doesn't actually receives data, caller needs to pass received UDP data into `Recv` function and RUDP object will ensure that messages are received in order.

## Algorithm Introduction

### Decoupling with UDP

The best part about this design is that it is entirely decoupled with UDP sending & receiving. Unlike many other packages which also ensure reliable transmission, instead, this package only provides the reliable transimission mechanism, while the caller needs to utilize its interfaces to inject data received from UDP and send processed data out with UDP.

This decoupling makes it very easy to integrate this package with any of an existing network framework.

### Provider & Consumer

The transmission is bidirectional. Let's define two roles in the case of a message transmission

- Provider: provides and sends a message

- Consumer: receives and handles a message

RUDP object is both a provider & consumer. When RUDP is sending message out, it is provider; when receiving message from other side, it is consumer.

## Message ID

Each message is with a uint16 ID, starting from 0. As we need to circulate the id when it reaches 0xffff, a function called `compareID` is implemented for the case when id reaches max uint16.

For example, if an old id is 0xfffe, a new id is 3, 0xfffe is larger than 3, but here we know that 3 is newer than 0xfffe, so `compareID` function ensures that 3 is larger than 0xfffe in our case. This makes sure that these IDs are still in correct order in this corner case.

This is different from the original C implementation, for simplicity.

## Unit Test

With the excellent tool of Go unit testing, the package is 100% unit test covered.

You can check out the unit test for detailed use of this package.
