//go:build freebsd

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/zombocoder/go-freebsd-libzfs/zpool"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("Usage: %s <operation> <pool-name> [options]\n", os.Args[0])
		fmt.Println("Operations:")
		fmt.Println("  status <pool-name>     - Show detailed pool status")
		fmt.Println("  export <pool-name>     - Export a pool")
		fmt.Println("  export-force <pool-name> - Force export a pool")
		fmt.Println("  import <pool-name>     - Import a pool")
		fmt.Println("  destroy <pool-name>    - Destroy a pool (DANGEROUS!)")
		fmt.Println("")
		fmt.Println("WARNING: Some operations are destructive and irreversible!")
		os.Exit(1)
	}

	operation := os.Args[1]
	poolName := os.Args[2]
	
	ctx := context.Background()

	client, err := zpool.New(ctx)
	if err != nil {
		log.Fatalf("Failed to create pool client: %v", err)
	}
	defer client.Close()

	switch operation {
	case "status":
		err = showPoolStatus(ctx, client, poolName)
	case "export":
		err = exportPool(ctx, client, poolName, false)
	case "export-force":
		err = exportPool(ctx, client, poolName, true)
	case "import":
		err = importPool(ctx, client, poolName)
	case "destroy":
		err = destroyPool(ctx, client, poolName)
	default:
		log.Fatalf("Unknown operation: %s", operation)
	}

	if err != nil {
		log.Fatalf("Operation failed: %v", err)
	}

	fmt.Printf("Operation '%s' on pool '%s' completed successfully\n", operation, poolName)
}

func showPoolStatus(ctx context.Context, client *zpool.Client, poolName string) error {
	fmt.Printf("Getting status for pool: %s\n", poolName)

	status, err := client.GetStatus(ctx, poolName)
	if err != nil {
		return fmt.Errorf("failed to get pool status: %w", err)
	}

	fmt.Printf("\nPool Status:\n")
	fmt.Printf("============\n")
	fmt.Printf("Name: %s\n", status.Pool.Name)
	fmt.Printf("GUID: %d\n", status.Pool.GUID)
	fmt.Printf("Health: %s\n", status.Pool.Health)
	fmt.Printf("State: %s\n", status.Pool.State)

	// Try to get some properties as well
	props, err := client.GetProperties(ctx, poolName, "size", "free", "allocated", "capacity")
	if err == nil && len(props) > 0 {
		fmt.Printf("\nProperties:\n")
		fmt.Printf("-----------\n")
		for name, prop := range props {
			fmt.Printf("%-12s: %v (source: %s)\n", name, prop.Value, prop.Source)
		}
	}

	return nil
}

func exportPool(ctx context.Context, client *zpool.Client, poolName string, force bool) error {
	fmt.Printf("Exporting pool: %s (force: %v)\n", poolName, force)
	
	// Confirm the operation
	if !force {
		fmt.Print("Are you sure you want to export this pool? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			return fmt.Errorf("operation cancelled by user")
		}
	}

	opts := zpool.ExportOptions{
		Force:   force,
		Message: "Exported via Go ZFS library example",
	}

	return client.Export(ctx, poolName, opts)
}

func importPool(ctx context.Context, client *zpool.Client, poolName string) error {
	fmt.Printf("Importing pool: %s\n", poolName)

	opts := zpool.ImportOptions{
		Force: false, // Don't force by default for safety
	}

	return client.Import(ctx, poolName, opts)
}

func destroyPool(ctx context.Context, client *zpool.Client, poolName string) error {
	fmt.Printf("WARNING: You are about to DESTROY pool '%s'\n", poolName)
	fmt.Println("This operation is IRREVERSIBLE and will DELETE ALL DATA!")
	fmt.Print("Type 'DELETE' to confirm: ")
	
	var confirmation string
	fmt.Scanln(&confirmation)
	
	if confirmation != "DELETE" {
		return fmt.Errorf("operation cancelled - confirmation not provided")
	}

	fmt.Printf("Destroying pool: %s\n", poolName)
	return client.Destroy(ctx, poolName)
}