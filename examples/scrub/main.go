//go:build freebsd

// Command scrub controls and monitors pool scrubs.
//
// Usage:
//
//	scrub start [-w] <pool>   # start or resume a scrub; -w waits for it to finish
//	scrub pause <pool>        # pause a running scrub
//	scrub stop <pool>         # cancel a scrub
//	scrub status <pool>       # show scrub progress
//
// start, pause and stop require root.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/zombocoder/go-freebsd-libzfs/zpool"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s start [-w] <pool> | pause <pool> | stop <pool> | status <pool>\n", os.Args[0])
	os.Exit(2)
}

func main() {
	if len(os.Args) < 3 {
		usage()
	}
	cmd := os.Args[1]
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	wait := fs.Bool("w", false, "wait for the scrub to finish (start only)")
	_ = fs.Parse(os.Args[2:])
	if fs.NArg() != 1 {
		usage()
	}
	pool := fs.Arg(0)

	// Ctrl-C stops waiting; it does not cancel the scrub itself.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	client, err := zpool.New(ctx)
	if err != nil {
		log.Fatalf("Failed to create pool client: %v", err)
	}
	defer client.Close()

	switch cmd {
	case "start":
		if err := client.StartScrub(ctx, pool); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("scrub started on %s\n", pool)
		if *wait {
			waitForScrub(ctx, client, pool)
		}
	case "pause":
		if err := client.PauseScrub(ctx, pool); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("scrub paused on %s\n", pool)
	case "stop":
		if err := client.StopScrub(ctx, pool); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("scrub stopped on %s\n", pool)
	case "status":
		scan := scanStatus(ctx, client, pool)
		printScan(pool, scan)
	default:
		usage()
	}
}

func scanStatus(ctx context.Context, client *zpool.Client, pool string) *zpool.ScanStatus {
	st, err := client.GetStatus(ctx, pool)
	if err != nil {
		log.Fatal(err)
	}
	return st.Scan
}

func printScan(pool string, s *zpool.ScanStatus) {
	if s == nil {
		fmt.Printf("%s: no scrub or resilver has been run\n", pool)
		return
	}
	fmt.Printf("%s: %s %s, started %s\n", pool, s.Function, s.State, s.StartTime.Format(time.RFC1123))
	inProgress := s.State == zpool.ScanStateScanning || s.State == zpool.ScanStateSuspended
	if inProgress && s.ToExamine > 0 {
		fmt.Printf("  issued %d of %d bytes (%.2f%%)", s.Issued, s.ToExamine,
			100*float64(s.Issued)/float64(s.ToExamine))
		if s.Rate > 0 {
			fmt.Printf(" at %d B/s", s.Rate)
		}
		fmt.Println()
	}
	if s.EndTime != nil {
		fmt.Printf("  ended %s, took %s\n", s.EndTime.Format(time.RFC1123),
			s.EndTime.Sub(s.StartTime).Round(time.Second))
	}
	fmt.Printf("  repaired %d bytes, %d errors\n", s.Processed, s.Errors)
}

func waitForScrub(ctx context.Context, client *zpool.Client, pool string) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	progressShown := false
	for {
		scan := scanStatus(ctx, client, pool)
		if scan != nil && scan.State != zpool.ScanStateScanning {
			if progressShown {
				fmt.Println()
			}
			printScan(pool, scan)
			if scan.Errors > 0 {
				os.Exit(1)
			}
			return
		}
		if scan != nil && scan.ToExamine > 0 {
			fmt.Printf("\r  %.2f%% done", 100*float64(scan.Issued)/float64(scan.ToExamine))
			progressShown = true
		}
		select {
		case <-ctx.Done():
			fmt.Println("\nstopped waiting; the scrub continues in the background")
			return
		case <-ticker.C:
		}
	}
}
