package main

import (
    "flag"
    "fmt"
    "os"
    "log"
    "regexp"

    "github.com/go-stomp/stomp"
    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
)

const defaultPort = ":61613"

var serverAddr = flag.String("server", "localhost:61613", "STOMP server endpoint")
var messageCount = flag.Int("count", 1, "Number of messages to send/receive")
var queueName = flag.String("queue", "/queue/INT.CANONICAL.GeolocationFrame.2.1", "ActiveMQ queue name")
var helpFlag = flag.Bool("help", false, "Print help text")
var stop = make(chan bool)

var mongoUri = flag.String("mongoUri", "localhost:27017", "MongoDB URI")
var mongoDb = flag.String("mongoDb", "customers_shared", "Mongo database to persist messages to")
var mongoCollection = flag.String("mongoCollection", "amq_cep_replay", "Mongo collection to persist messages to")
var mongoQuery = flag.String("query", "{}", "Mongo query to filter messages")

var recvFlag = flag.Bool("recv", false, "Read messages from AMQ")
var sendFlag = flag.Bool("send", false, "Send messages to AMQ")

// these are the default options that work with RabbitMQ
var options []func(*stomp.Conn) error = []func(*stomp.Conn) error{
    stomp.ConnOpt.Login("guest", "guest"),
    stomp.ConnOpt.Host("/"),
}

/*
    Implementation specific
*/
/*
type Location struct {
    latitude float64 ",omitempty"

    longitude float64 ",omitempty"

    numberOfSatellite int ",omitempty"

    hdop float32 ",omitempty"

    poiReference string ",omitempty"
}

type Movement struct {
    speed float32 ",omitempty"

    direction float32 ",omitempty"

}
type Journey struct {
    gpsDistance float32 ",omitempty"

    canDistance float32 ",omitempty"

    gpsOrCanDistance float32 ",omitempty"

    fuelUsed float32 ",omitempty"

}
type AssetInformation struct {
    odometer float32 ",omitempty"

    fuelLevel float32 ",omitempty"

    totalFuelUsed float32 ",omitempty"

    batteryLevel float32 ",omitempty"

    kmsToService float32 ",omitempty"

    daysToService float32 ",omitempty"

}

type Frame struct {
    assetId string
    assetType string ",omitempty"

    deviceId string ",omitempty"

    imei string ",omitempty"

    esn int64 ",omitempty"

    simId string ",omitempty"

    legacyId string ",omitempty"

    clientId string

    driverId string ",omitempty"

    driverKey string ",omitempty"

    driverType string ",omitempty"

    driverSource string ",omitempty"

    privacy bool ",omitempty"

    obuGenerationDate string ",omitempty"

    gatewayReceptionDate string ",omitempty"

    obuSequenceId int64 ",omitempty"

    gatewaySequenceId int64 ",omitempty"

    index int ",omitempty"

    eventCategory int ",omitempty"

    eventCode int ",omitempty"

    location Location ",omitempty"

    movement Movement ",omitempty"

    journey []Journey ",omitempty"

    assetInformation AssetInformation ",omitempty"

}
*/
type Frame interface{}

type FrameFromAmq struct {
    Ordinal int
    Frame
}

func help() {
    fmt.Fprintf(os.Stderr, "Usage of %s\n", os.Args[0])
    flag.PrintDefaults()
    os.Exit(1)
}

func main() {
    flag.Parse()
    if *helpFlag {
        help()
    }

    if *recvFlag {
        log.Print("Mode: receive")
        recvMessages()
    } else if *sendFlag {
        log.Print("Mode: send")
        sendMessages()
   } else {
        help()
    }
}

func sendMessages() {

    conn, err := stomp.Dial("tcp", *serverAddr, options...)
    if err != nil {
        log.Fatal("cannot connect: ", err)
        return
    }

    c := mongoConnect()
    log.Print(c)

    result := FrameFromAmq{}
    var query interface{}
//"{\"frame.assetId\": \"554994\"}"
    err = bson.UnmarshalJSON([]byte(*mongoQuery), &query)
    if err != nil {
        log.Fatal(err)
    }
 

    iter := c.Find(query).Sort("ordinal").Limit(*messageCount).Iter()
    
    for iter.Next(&result) {

//        log.Print("raw: ", result.Frame)

        json, err2 := bson.MarshalJSON(result.Frame)
        if err2 != nil {
            log.Fatal(err2)
        }
        log.Print("result: ", result.Frame)
        err = conn.Send(*queueName, "text/plain",
            []byte(string(json)))
        if err != nil {
            log.Fatal(err)
        }
 
    }
    if err := iter.Close(); err != nil {
        log.Fatal(err)
    }

    conn.Disconnect()
    log.Print("sender finished")
}

func recvMessages() {

    conn, err := stomp.Dial("tcp", *serverAddr, options...)

    if err != nil {
        log.Fatal("cannot connect: ", err)
        return
    }

    sub, err := conn.Subscribe(*queueName, stomp.AckAuto)
    if err != nil {
        log.Fatal("cannot subscribe to", *queueName, err)
        return
    }
    //----

    c := mongoConnect()

    //----
    for i := 1; i <= *messageCount; i++ {
        msg := <-sub.C
        actualText := string(msg.Body)

        actualBytes := []byte(actualText)
        regexpFix, errR := regexp.Compile("\" :")
        if errR != nil {
            log.Fatal(err)
        }

        correctedBytes := regexpFix.ReplaceAllLiteral(actualBytes, []byte("\":"))

        var bdoc Frame
        err = bson.UnmarshalJSON(correctedBytes, &bdoc)
        if err != nil {
            log.Fatal(err)
        }

        err = c.Insert(&FrameFromAmq{i, bdoc})

    }

    log.Print("receiver finished")

}

func mongoConnect() *mgo.Collection {
    log.Print("Connecting to MongoDB at ", *mongoUri)

    session, err := mgo.Dial(*mongoUri)
    if err != nil {
        panic(err)
    }
//    defer session.Close()

    // Optional. Switch the session to a monotonic behavior.
//    session.SetMode(mgo.Monotonic, true)

    return session.DB(*mongoDb).C(*mongoCollection)

}
 
