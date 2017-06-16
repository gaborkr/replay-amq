
Summary
=======

This is a small tool to capture messages from AMQ and save to MongoDB, and to query messages from MongoDB and replay them to AMQ.

The real power of this small tool is that you can drive message replay by a MongoDB query, so you can be very selective about what to replay.

Setup
=====

You will need the Go tools (https://golang.org/doc/install) to compile this program.

However, since it is compiled to a single static binary, once you have it compiled you can just pass it on as a Windows .exe for example.

Other advantages include small size and minimal memory consumption.

Compile
-------

In your Go workspace directory (default: $HOME/go) `mkdir src; cd src`, then `git clone` this project.

Compile with `go build` - you will get an executable in the same dir. It is a console app, run it and it will present you with the options.

Usage
=====

You need ActiveMQ and MongoDB up and running somewhere - could be your local machine, or a dev/test environment. The default values for the configuration all point to localhost.

Configuration
-------------

Default values are given below.

ActiveMQ
--------

* `-server "localhost:61613"` ActiveMQ broker instance, with STOMP enabled

* `-queue "/queue/INT.CANONICAL.GeolocationFrame.2.1"` queue or topic the messages should be consumed from

MongoDB
-------

* `-mongoUri "localhost:27017"` MongoDb host:port

* `-mongoDb "customers_shared"` MongoDb database

* `-mongoCollection "amq_cep_replay"` MongoDb collection

Capture
=======

* `./replay-amq -recv` will capture one message from a queue - useful for testing your configuration

* `./replay-amq -recv -count 999999999` will start capturing "indefinitely" - useful if you want to keep capturing a large number of messages or until a certain time. Can be stopped with Ctrl-C.

The format of the messages captured in MongoDB is like this:

```
{
    "_id" : ObjectId("592d3a2d796035974e9b3bae"),
    "ordinal" : 1,
    "frame" : {
        ...
    }
}
```

The ordinal is the sequence number of the messages captured, frame contains the original message.

Replay
======

Once you have captured some messages, you can replay them to the same queue (or another):

* `./replay-amq -send` will replay the first message from the capture. The message will NOT be deleted from MongoDB!

* `./replay-amq -send -count 10 -query '{"ordinal": {"$gt": 20}}'` will replay 10 messages starting from message 20 (make sure you escape the quotes as required by your shell!). Note that the query is not limited to the metadata, you can of course do a query based on the data within the captured frame!

Replay will send messages without delay.

