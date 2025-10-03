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
	if len(os.Args) < 3 {
		fmt.Printf("Usage: %s <operation> <dataset-name> [options]\n", os.Args[0])
		fmt.Println("Operations:")
		fmt.Println("  create-fs <dataset-name>     - Create a filesystem")
		fmt.Println("  create-vol <dataset-name> <size> - Create a volume (e.g., 1G, 500M)")
		fmt.Println("  destroy <dataset-name>       - Destroy a dataset")
		fmt.Println("  destroy-recursive <dataset-name> - Destroy dataset and children")
		fmt.Println("  info <dataset-name>          - Show dataset information")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  create-fs zroot/test         - Create filesystem zroot/test")
		fmt.Println("  create-vol zroot/vol1 1G     - Create 1GB volume zroot/vol1")
		fmt.Println("  destroy zroot/test           - Destroy zroot/test")
		fmt.Println("")
		fmt.Println("WARNING: Destroy operations are irreversible!")
		os.Exit(1)
	}

	operation := os.Args[1]
	datasetName := os.Args[2]
	
	ctx := context.Background()

	client, err := zfs.New(ctx)
	if err != nil {
		log.Fatalf("Failed to create dataset client: %v", err)
	}
	defer client.Close()

	switch operation {
	case "create-fs":
		err = createFilesystem(ctx, client, datasetName)
	case "create-vol":
		if len(os.Args) < 4 {
			log.Fatalf("Volume size required for create-vol operation")
		}
		size := os.Args[3]
		err = createVolume(ctx, client, datasetName, size)
	case "destroy":
		err = destroyDataset(ctx, client, datasetName, false)
	case "destroy-recursive":
		err = destroyDataset(ctx, client, datasetName, true)
	case "info":
		err = showDatasetInfo(ctx, client, datasetName)
	default:
		log.Fatalf("Unknown operation: %s", operation)
	}

	if err != nil {
		log.Fatalf("Operation failed: %v", err)
	}

	fmt.Printf("Operation '%s' on dataset '%s' completed successfully\n", operation, datasetName)
}

func createFilesystem(ctx context.Context, client *zfs.Client, datasetName string) error {
	fmt.Printf("Creating filesystem: %s\n", datasetName)

	properties := map[string]string{
		"compression": "lz4",
		"atime":       "off",
	}

	return client.CreateFilesystem(ctx, datasetName, properties)
}

func createVolume(ctx context.Context, client *zfs.Client, datasetName, size string) error {
	fmt.Printf("Creating volume: %s (size: %s)\n", datasetName, size)

	properties := map[string]string{
		"compression": "lz4",
	}

	return client.CreateVolume(ctx, datasetName, size, properties)
}

func destroyDataset(ctx context.Context, client *zfs.Client, datasetName string, recursive bool) error {
	fmt.Printf("WARNING: You are about to DESTROY dataset '%s'\n", datasetName)
	if recursive {
		fmt.Println("This will RECURSIVELY destroy ALL child datasets!")
	}
	fmt.Println("This operation is IRREVERSIBLE and will DELETE ALL DATA!")
	fmt.Print("Type 'DELETE' to confirm: ")
	
	var confirmation string
	fmt.Scanln(&confirmation)
	
	if confirmation != "DELETE" {
		return fmt.Errorf("operation cancelled - confirmation not provided")
	}

	fmt.Printf("Destroying dataset: %s (recursive: %v)\n", datasetName, recursive)
	return client.Destroy(ctx, datasetName, recursive)
}

func showDatasetInfo(ctx context.Context, client *zfs.Client, datasetName string) error {
	fmt.Printf("Getting information for dataset: %s\n", datasetName)

	// Get basic dataset info
	dataset, err := client.Get(ctx, datasetName)
	if err != nil {
		return fmt.Errorf("failed to get dataset: %w", err)
	}

	fmt.Printf("\nDataset Information:\n")
	fmt.Printf("====================\n")
	fmt.Printf("Name: %s\n", dataset.Name)
	fmt.Printf("Type: %s\n", dataset.Type)
	fmt.Printf("GUID: %d\n", dataset.GUID)
	fmt.Printf("Pool: %s\n", dataset.Pool())
	fmt.Printf("Is Snapshot: %v\n", dataset.IsSnapshot())
	fmt.Printf("Is Filesystem: %v\n", dataset.IsFilesystem())
	fmt.Printf("Is Volume: %v\n", dataset.IsVolume())

	// Get properties
	props, err := client.GetProperties(ctx, datasetName, "used", "avail", "mountpoint", 
		"compression", "compressratio", "quota", "reservation")
	if err == nil && len(props) > 0 {
		fmt.Printf("\nProperties:\n")
		fmt.Printf("-----------\n")
		for name, prop := range props {
			fmt.Printf("%-15s: %v (source: %s)\n", name, prop.Value, prop.Source)
		}
	}

	// If it's a filesystem, try to get mount-related properties
	if dataset.IsFilesystem() {
		fmt.Printf("\nFilesystem-specific info:\n")
		fmt.Printf("-------------------------\n")
		
		if mountpoint, err := client.GetStringProperty(ctx, datasetName, "mountpoint"); err == nil {
			fmt.Printf("Mountpoint: %s\n", mountpoint)
		}
		
		if canmount, err := client.GetStringProperty(ctx, datasetName, "canmount"); err == nil {
			fmt.Printf("Can mount: %s\n", canmount)
		}
		
		if mounted, err := client.GetStringProperty(ctx, datasetName, "mounted"); err == nil {
			fmt.Printf("Currently mounted: %s\n", mounted)
		}
	}

	// If it's a volume, try to get volume-specific properties
	if dataset.IsVolume() {
		fmt.Printf("\nVolume-specific info:\n")
		fmt.Printf("---------------------\n")
		
		if volsize, err := client.GetStringProperty(ctx, datasetName, "volsize"); err == nil {
			fmt.Printf("Volume size: %s\n", volsize)
		}
		
		if volblocksize, err := client.GetStringProperty(ctx, datasetName, "volblocksize"); err == nil {
			fmt.Printf("Volume block size: %s\n", volblocksize)
		}
	}

	return nil
}