package utils

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
)

// CrackAuthenticator attempts to brute force the 8-digit key from an authenticator hex string
// using highly concurrent Goroutines to maximize CPU utilization.
func CrackAuthenticator(ctx context.Context, authenticator string) (string, error) {
	if len(authenticator) < 10 {
		return "", errors.New("invalid authenticator length")
	}

	numWorkers := runtime.NumCPU()
	if numWorkers < 4 {
		numWorkers = 4 // At least 4 workers
	}

	totalKeys := 100000000 // Test all 8 digits: 00000000 to 99999999
	chunkSize := totalKeys / numWorkers

	// Create a cancelable context to stop all workers once the key is found
	ctxCancel, cancel := context.WithCancel(ctx)
	defer cancel()

	resultChan := make(chan string, 1)
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			start := workerID * chunkSize
			end := start + chunkSize
			if workerID == numWorkers-1 {
				end = totalKeys // Ensure the last worker covers any remainder
			}

			for x := start; x < end; x++ {
				// Check if another worker found the key or context was cancelled
				select {
				case <-ctxCancel.Done():
					return
				default:
				}

				key := fmt.Sprintf("%08d", x)
				crypto := NewTripleDESCrypto(key)

				// Attempt decryption
				decodedText, err := crypto.ECBDecrypt(authenticator)
				if err != nil {
					continue
				}

				// The decrypted plaintext should contain multiple '$' separated fields
				infos := strings.Split(decodedText, "$")
				if len(infos) > 7 {
					// We found the correct key!
					select {
					case resultChan <- key:
						cancel() // Stop all other workers immediately
					default:
					}
					return
				}
			}
		}(i)
	}

	// Close the channel once all workers complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	select {
	case key, ok := <-resultChan:
		if ok {
			return key, nil
		}
		return "", errors.New("failed to crack the key in the search space")
	case <-ctx.Done():
		return "", ctx.Err()
	}
}
