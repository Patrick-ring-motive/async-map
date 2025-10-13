package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Patrick-ring-motive/async-map/asyncmap" // Replace 'github.com/youruser/async-map' with your actual module path
)

// Example struct for demonstration
type User struct {
	ID   int
	Name string
}

func main() {
	log.Println("--- Testing asyncmap Package ---")

	// 1. Initialization using NewSyncMap (Preferred, efficient)
	log.Println("1. Initialize with NewSyncMap")
	userMap := asyncmap.NewSyncMap[int, User](map[int]User{
		101: {ID: 101, Name: "Alice"},
		102: {ID: 102, Name: "Bob"},
	})

	// 2. Testing Store and Load
	userMap.Store(103, User{ID: 103, Name: "Charlie"})
	if u, ok := userMap.Load(101); ok {
		log.Printf("Loaded User 101: %+v\n", u)
	}

	// 3. Testing GetOrDefault
	keyToFind := 104
	defaultUser := User{ID: 999, Name: "Guest"}
	uDefault := userMap.GetOrDefault(keyToFind, defaultUser)
	log.Printf("GetOrDefault for key %d: %+v (Key was not found)\n", keyToFind, uDefault)

	// 4. Testing Zero-Value Initialization (lazyInit)
	log.Println("4. Testing Zero-Value Initialization (Lazy Init)")
	var settingsMap asyncmap.SyncMap[string, time.Duration]
	settingsMap.Store("timeout", 5*time.Second) // This call triggers lazyInit
	settingsVal := settingsMap.Get("timeout")
	log.Printf("Lazy Init Test: Stored 'timeout', retrieved: %v\n", settingsVal)

	// 5. Testing Range
	log.Println("5. Testing Range (Iteration and Safety)")
	userMap.Range(func(id int, user User) bool {
		log.Printf("  Map Entry: ID=%d, Name=%s\n", id, user.Name)
		return true
	})

	// 6. Demonstrate Merge and Transform
	mapB := asyncmap.NewSyncMap[int, User](map[int]User{
		102: {ID: 102, Name: "NewBob"}, // Overwrites existing
		105: {ID: 105, Name: "Eve"},
	})

	mergedMap := asyncmap.Merge(userMap, mapB)
	log.Printf("6. Merged Map Size: %d\n", len(mergedMap.ToMap()))
	
	// Transform map to string key/int value map
	transformedMap := asyncmap.SyncTransform(userMap, func(k int, v User) (string, int) {
		return fmt.Sprintf("User_%d", k), v.ID
	})
	log.Printf("7. Transformed Map Keys: %+v\n", transformedMap.ToMap())
}
