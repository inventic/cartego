cartego
=======

**LEGAL DISCLAIMER:** Cartego is not affiliated with any of the maps services
it supports. [Open Street Maps](http://www.openstreetmap.org/), however,
is *open data*, and as such is the only officially supported provider, with the
other services provided for completeness. Cartego does not offer any rights or
protections outside of the terms and conditions of each respective service.
Please note that use of this software to download map imagery from a maps service
may be illegal in your area.

Cartego is a library for downloading tiles from popular maps services. 
Cartego is loosely based on [cartegan](https://github.com/SpotterRF/cartegan),
but cartego does offer some distinct differences:

* cartego provides a server (and protocol) for hosting map tiles
* cartego is written in Go
  * easy to embed in Go applications
  * users only need a binary to run stand-alone
* cartego is a library (easy to embed) and a protocol built on HTTP

License
=======

Cartego is licensed under the Apache License 2.0.
