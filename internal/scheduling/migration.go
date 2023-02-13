package scheduling

import (
	"fmt"
	"time"

	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/container"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/node"
)

// Start the thread that monitors the node and decides whether a migration process is necessary
func startMigrationMonitor() {
	// Acquire the node total memory and the threshold above which migration should occur
	totalMemory := config.GetFloat(config.POOL_MEMORY_MB, 1024)
	threshold := getThreshold()
	for true {
		// Acquire the available memory and check if it is over the threshold
		availableMemory := float64(node.Resources.AvailableMemMB)
		percMem := (totalMemory - availableMemory) / totalMemory
		fmt.Println("MEMORY_MB: ", totalMemory-availableMemory, " / ", totalMemory, " = ", percMem*100, "%")
		if availableMemory <= (1-threshold)*(totalMemory) {
			// Select the best container candidate to migrate
			fmt.Println("Node load threshold exceeded. Selecting the best container to migrate...")
			migrateAContainer()
		}
		time.Sleep(2 * time.Second)
	}
}

// Select the container and the node to migrate to
func migrateAContainer() {
	var migrationNodeCandidates []string
	var containerToMigrate string
	// TODO: define an algorithm to find the best node candidates to migrate a container
	// TODO: define an algorithm to find the best container candidate to migrate
	migrationNodeCandidates = getMigrationTargetNodes()
	containerToMigrate = getMigrationTargetContainer()

	Migrate(containerToMigrate, migrationNodeCandidates)
}

// Retrieve the original node ip (synchronous case, otherwise "")
func retrieveOriginalNodeIP() string {
	nodeIP := ""
	select {
	case nodeIP = <-migrationAddresses:
		return nodeIP
	case <-time.After(1 * time.Second):
		fmt.Println("A problem occurred trying to retrieve migrator client's IP. The result will be posted on ETCD.")
	}
	return nodeIP
}

// Define the node's memory usage threshold above which a migration will occur
func getThreshold() float64 {
	defaultThreshold := 0.8
	thr := config.GetFloat(config.MIGRATION_DECISION_THRESHOLD, defaultThreshold)
	if thr < 0 || thr > 1 {
		return defaultThreshold
	} else {
		return thr
	}
}

// TODO: define an algorithm to find the best node candidates to migrate a container
func getMigrationTargetNodes() []string {
	return []string{"34.77.63.161"}
}

// TODO: define an algorithm to find the best container candidate to migrate
func getMigrationTargetContainer() string {
	// At the moment a random container associated with the least priority is selected
	var containerToMigrate string
	for contID, _ := range node.NodeRequests {
		// Take the first container on the node
		containerToMigrate = contID
		break
	}
	// Register its priority
	minimumPriority := node.NodeRequests[containerToMigrate].OriginalRequest.Class
	requestToMigrate := node.NodeRequests[containerToMigrate]
	for contID, r := range node.NodeRequests {
		// Look for the least priority container on the node
		if r.OriginalRequest.Class <= minimumPriority {
			requestToMigrate = r
			minimumPriority = r.OriginalRequest.Class
			containerToMigrate = contID
		}
	}
	requestToMigrate.OriginalRequest.ExecReport.Migrated = true /* Necessary: set this field to true before migrating.
	This will allow the node API to know if the result will come normally or if it has to be polled from ETCD
	*/
	return containerToMigrate
}

// Demo function to trigger the migraion manually
// TODO - remove this function, use the monitoring thread instead
func migration_demo(request *function.Request, containerID container.ContainerID) {
	fallbackAddresses := []string{"34.77.63.161"} // Manually set the migration addresses
	request.ExecReport.Migrated = true            // Necessary: set this field to true before migrating
	Migrate(containerID, fallbackAddresses)
}
