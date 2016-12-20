## About

This is a go transplant of a RUDP implementation, originally written in C by the lengendary developer cloudwu from China.

The original algorithm explaination is at [cloudwu's blog](http://blog.codingnow.com/2016/03/reliable_udp.html) (In Chinese)
and [here](https://github.com/cloudwu/rudp) is the source c code repo.

## Algorithm Introduction

### Provider & Consumer

- Provider: provides and sends a message
- Consumer: receives and handles a message

RUDP object is both a provider & consumer. When RUDP is sending message out, it is provider; when receiving message from other side, it is consumer.

### Decoupling with UDP

The best part about this design is that it is entirely decoupled with UDP sending & receiving. On the contrary, many other similar packages which also ensure reliable transmission are highly coupled with UDP.

This decoupling makes it very easy to integrate this package within any of an existing framework.

### Test Coverage

With the excellent tool of Go unit testing, the package is 100% unit test covered.