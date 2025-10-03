//go:build freebsd

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/zombocoder/go-freebsd-libzfs/zfs"
	"github.com/zombocoder/go-freebsd-libzfs/zpool"
	"github.com/zombocoder/go-freebsd-libzfs/errors"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== ZFS Error Handling Examples ===")

	// Test pool error handling
	fmt.Println("\n1. Testing pool error handling...")
	poolClient, err := zpool.New(ctx)
	if err != nil {
		log.Fatalf("Failed to create pool client: %v", err)
	}
	defer poolClient.Close()

	// Try to get a non-existent pool
	_, err = poolClient.Get(ctx, "nonexistent-pool-12345")
	if err != nil {
		fmt.Printf("Expected error for non-existent pool: %v\n", err)
		
		if errors.IsPoolNotFound(err) {
			fmt.Println("✓ Correctly detected as pool not found")
		} else {
			fmt.Println("✗ Failed to detect as pool not found")
		}

		// Check if it's a ZFS error
		if zfsErr, ok := errors.AsZfsError(err); ok {
			fmt.Printf("✓ ZFS Error details - Op: %s, Code: %s, Detail: %s\n", 
				zfsErr.Op, zfsErr.Code, zfsErr.Detail)
		}
	}

	// Test dataset error handling
	fmt.Println("\n2. Testing dataset error handling...")
	datasetClient, err := zfs.New(ctx)
	if err != nil {
		log.Fatalf("Failed to create dataset client: %v", err)
	}
	defer datasetClient.Close()

	// Try to get a non-existent dataset
	_, err = datasetClient.Get(ctx, "nonexistent/dataset/12345")
	if err != nil {
		fmt.Printf("Expected error for non-existent dataset: %v\n", err)
		
		if errors.IsDatasetNotFound(err) {
			fmt.Println("✓ Correctly detected as dataset not found")
		} else {
			fmt.Println("✗ Failed to detect as dataset not found")
		}

		// Check various error predicates
		fmt.Printf("Is dataset not found: %v\n", errors.IsDatasetNotFound(err))
		fmt.Printf("Is pool not found: %v\n", errors.IsPoolNotFound(err))
		fmt.Printf("Is permission denied: %v\n", errors.IsPermissionDenied(err))
		fmt.Printf("Is exists: %v\n", errors.IsExists(err))
		fmt.Printf("Is busy: %v\n", errors.IsBusy(err))
		fmt.Printf("Is no space: %v\n", errors.IsNoSpace(err))
		fmt.Printf("Is not supported: %v\n", errors.IsNotSupported(err))
		fmt.Printf("Is ZFS error: %v\n", errors.IsZfsError(err))
	}

	// Test property error handling
	fmt.Println("\n3. Testing property error handling...")
	
	// Try to get properties for non-existent dataset
	_, err = datasetClient.GetProperties(ctx, "fake/dataset", "used", "avail")
	if err != nil {
		fmt.Printf("Expected error for properties on non-existent dataset: %v\n", err)
		
		if errors.IsDatasetNotFound(err) {
			fmt.Println("✓ Property retrieval correctly failed with dataset not found")
		}
	}

	// Test custom error creation
	fmt.Println("\n4. Testing custom error creation...")
	
	customErr := errors.NewZfsError("test_operation", "test_resource", 
		errors.ErrCodeInval, 22, "custom test error", nil)
	fmt.Printf("Custom error: %v\n", customErr)
	
	wrappedErr := fmt.Errorf("operation failed: %w", customErr)
	if errors.IsZfsError(wrappedErr) {
		fmt.Println("✓ Wrapped error correctly detected as ZFS error")
	}

	// Test error wrapping
	baseErr := fmt.Errorf("underlying system error")
	wrappedZfsErr := errors.WrapZfsError("wrap_test", "test_resource", baseErr)
	fmt.Printf("Wrapped ZFS error: %v\n", wrappedZfsErr)

	fmt.Println("\n=== Error handling test completed ===")
}