//go:build freebsd

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/zombocoder/go-freebsd-libzfs/zfs"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <dataset-name>\n", os.Args[0])
		fmt.Println("Shows detailed properties for a ZFS dataset")
		os.Exit(1)
	}

	datasetName := os.Args[1]
	ctx := context.Background()

	client, err := zfs.New(ctx)
	if err != nil {
		log.Fatalf("Failed to create dataset client: %v", err)
	}
	defer client.Close()

	// Check if dataset exists
	dataset, err := client.Get(ctx, datasetName)
	if err != nil {
		log.Fatalf("Failed to get dataset %s: %v", datasetName, err)
	}

	fmt.Printf("Dataset: %s\n", dataset.Name)
	fmt.Printf("Type: %s\n", dataset.Type)
	fmt.Printf("GUID: %d\n", dataset.GUID)
	fmt.Printf("Pool: %s\n", dataset.Pool())
	fmt.Printf("Is Snapshot: %v\n", dataset.IsSnapshot())
	fmt.Printf("Is Filesystem: %v\n", dataset.IsFilesystem())
	fmt.Printf("Is Volume: %v\n", dataset.IsVolume())
	fmt.Printf("Is Bookmark: %v\n", dataset.IsBookmark())
	fmt.Println()

	// Get all available properties
	props, err := client.GetProperties(ctx, datasetName)
	if err != nil {
		log.Fatalf("Failed to get properties: %v", err)
	}

	fmt.Printf("Properties for dataset '%s':\n", datasetName)
	fmt.Println("=" + fmt.Sprintf("%*s", len(datasetName)+25, ""))

	for name, prop := range props {
		fmt.Printf("%-20s: %v (source: %s)\n", name, prop.Value, prop.Source)
	}

	// Test specific property getters
	fmt.Println("\nSpecific property tests:")
	
	if used, err := client.GetStringProperty(ctx, datasetName, "used"); err == nil {
		fmt.Printf("Used (string): %s\n", used)
	}
	
	if avail, err := client.GetStringProperty(ctx, datasetName, "avail"); err == nil {
		fmt.Printf("Available (string): %s\n", avail)
	}
	
	if mountpoint, err := client.GetStringProperty(ctx, datasetName, "mountpoint"); err == nil {
		fmt.Printf("Mountpoint (string): %s\n", mountpoint)
	}
}