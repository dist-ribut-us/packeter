## Dev Notes

Add test for timed out message

Instead of allocating the whole list upfront, use a map. A malicious attacker
could send a whole bunch of single packets belonging to messages with high
packet counts. That would clog up a bunch of memory quickly. Using the map will
only reserve the memory when a packet is actually received. Then we can put them
in order right before processing.

Currently, all packets in a message must come from the same address. That should
either be removed, or made optional. We may want to reused the packeter so that
a very large message can be broken down in to smaller pieces and sent through
many leap-routes.