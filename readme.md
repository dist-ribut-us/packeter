## Packeter
Takes large messages and breaks them down into small packets with parity shards
to provide a minimum guarentee that the recipient will be able to reassemble the
message, even if some packets are lost. Also handles the process of reassembling
the message as the packets are received.

[![GoDoc](https://godoc.org/github.com/dist-ribut-us/packeter?status.svg)](https://godoc.org/github.com/dist-ribut-us/packeter)