package main

import (
	"log"
	"net"
	"os"
	"time"

	"code.google.com/p/go-uuid/uuid"

	"flag"

	couchbasearray "github.com/andrewwebber/couchbase-array"
)

var servicePathFlag = flag.String("s", "/services/couchbase-array", "etcd directory")
var ttlFlag = flag.Int("ttl", 10, "time to live in seconds")
var debugFlag = flag.Bool("v", false, "verbose")
var processState = flag.Bool("p", true, "process state requests")
var machineIdentiferFlag = flag.String("ip", "", "machine ip address")

func main() {
	flag.Parse()

	machineIdentifier := *machineIdentiferFlag
	if machineIdentifier == "" {
		var err error
		machineIdentifier, err = getMachineIdentifier()
		if err != nil {
			log.Fatal(err)
		}
	}

	sessionID := uuid.New()

	for {
		announcments, err := couchbasearray.GetClusterAnnouncements(*servicePathFlag)
		if err != nil {
			panic(err)
		}

		machineState, ok := announcments[machineIdentifier]
		if !ok {
			machineState = couchbasearray.NodeState{machineIdentifier, sessionID, false, "", ""}
		}

		currentStates, err := couchbasearray.GetClusterStates(*servicePathFlag)

		if err == nil {
			if state, ok := currentStates[machineIdentifier]; ok {
				if state.State != machineState.State {
					log.Printf("DesiredState: %s - Current State: %s", state.DesiredState, machineState.State)
					machineState.State = "transitioning"
					if *processState {
						switch state.DesiredState {
						case couchbasearray.SchedulerStateClustered:
							log.Println("cluster")
						case couchbasearray.SchedulerStateNew:
							log.Println("init")
						}
					}
				}
			}
		}

		couchbasearray.SetClusterAnnouncement(*servicePathFlag, machineState)

		time.Sleep(time.Duration(*ttlFlag) * time.Second)
	}
}

func getMachineIdentifier() (string, error) {

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatalln(err)
	}

	var result string
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				result = ipnet.IP.String()
				log.Println(ipnet.Network())
				log.Printf("Found IP %s\n", result)
			}
		}
	}

	if result != "" {
		return result, nil
	}

	return os.Hostname()
}
