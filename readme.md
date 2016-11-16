## Packeter
Takes large messages and breaks them down into small packets with parity shards
to provide a minimum guarentee that the recipient will be able to reassemble the
message, even if some packets are lost. Also handles the process of reassembling
the message as the packets are received.

See [documentation](https://godoc.org/github.com/dist-ribut-us/packeter).