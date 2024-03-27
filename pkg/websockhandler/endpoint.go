package websockhandler

import (
	"fmt"
	"os"
	"sync"

	"github.com/gofrs/uuid"
)

var gPortsAvailable = map[int]uuid.UUID{}
var gPortsUse = map[uuid.UUID]int{}
var lockAvailable = sync.RWMutex{}
var lockUse = sync.RWMutex{}

// range of avaiable ports
// if anyone wants to change this, need to change the open ports range from the k8s deployment as well.
const (
	minPort = 10000
	maxPort = 10040
)

func endpointLocalGet(referenceID uuid.UUID) string {

	// get pod ip and available port
	localIP := os.Getenv("POD_IP")
	localPort := portGet(referenceID)
	res := fmt.Sprintf("%s:%d", localIP, localPort)

	return res
}

func endpointLocalRelease(referenceID uuid.UUID) {
	portRelease(referenceID)
}

func endpointInit() {
	lockAvailable.Lock()
	defer lockAvailable.Unlock()

	for i := minPort; i < maxPort; i++ {
		gPortsAvailable[i] = uuid.Nil
	}
}

func portGet(referenceID uuid.UUID) int {
	lockAvailable.Lock()
	defer lockAvailable.Unlock()

	for port := minPort; port < maxPort; port++ {
		if gPortsAvailable[port] == uuid.Nil {
			gPortsAvailable[port] = referenceID

			lockUse.Lock()
			defer lockUse.Unlock()
			gPortsUse[referenceID] = port

			return port
		}
	}

	return -1
}

func portRelease(referenceID uuid.UUID) {
	lockAvailable.Lock()
	defer lockAvailable.Unlock()

	lockUse.Lock()
	defer lockUse.Unlock()

	port := gPortsUse[referenceID]
	if port == 0 {
		// nothing to do
		return
	}
	delete(gPortsUse, referenceID)

	gPortsAvailable[port] = uuid.Nil
}
