## About

This is a go transplant of a RUDP implementation from the lengendary developer cloudwu from China.

The original algorithm explaination is at [cloudwu's blog](http://blog.codingnow.com/2016/03/reliable_udp.html) (In Chinese)
and [here](https://github.com/cloudwu/rudp) is the source c code repo.

## Algorithm Introduction

### Provider & Consumer

- Provider: provides and sends a message
- Consumer: receives and handles a message

RUDP is both provider & consumer. When RUDP is sending message out, it is provider; when receiving message from other side, it is consumer.