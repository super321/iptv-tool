package utils

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// CrackMode defines the brute-force key space
type CrackMode string

const (
	CrackModeDecimal CrackMode = "decimal" // 00000000 ~ 99999999  (10^8 = 100,000,000)
	CrackModeHex     CrackMode = "hex"     // 00000000 ~ FFFFFFFF  (16^8 = 4,294,967,296)
)

// CrackProgress holds real-time brute-force progress
type CrackProgress struct {
	Tried   int64   `json:"tried"`
	Total   int64   `json:"total"`
	Percent float64 `json:"percent"`
}

// CrackField represents a single named field from the decrypted authenticator
type CrackField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// CrackResult holds the cracked key and decrypted authenticator fields
type CrackResult struct {
	Key    string       `json:"key"`
	Fields []CrackField `json:"fields"`
}

// authenticator field names in order (format: random$EncryptToken$UserID$STBID$IP$MAC$Reserved$CTC)
var authenticatorFieldNames = []string{
	"Random", "EncryptToken", "UserID", "STBID", "IP", "MAC", "Reserved", "CTC",
}

// parseAuthenticatorFields splits the decrypted plaintext by '$' and maps to named fields
func parseAuthenticatorFields(plaintext string) []CrackField {
	parts := strings.Split(plaintext, "$")
	fields := make([]CrackField, len(authenticatorFieldNames))
	for i, name := range authenticatorFieldNames {
		val := ""
		if i < len(parts) {
			val = parts[i]
		}
		fields[i] = CrackField{Name: name, Value: val}
	}
	return fields
}

// CrackAuthenticator attempts to brute force the 8-character key from an authenticator hex string.
// It supports both decimal (00000000-99999999) and hex (00000000-FFFFFFFF) key spaces.
// progressCb is called approximately once per second with current progress. It may be nil.
func CrackAuthenticator(ctx context.Context, authenticator string, mode CrackMode, progressCb func(CrackProgress)) (*CrackResult, error) {
	if len(authenticator) < 10 {
		return nil, errors.New("invalid authenticator length")
	}

	var totalKeys int64
	switch mode {
	case CrackModeHex:
		totalKeys = 0x100000000 // 4,294,967,296
	default:
		totalKeys = 100000000 // 10^8
	}

	numWorkers := runtime.NumCPU()
	if numWorkers < 4 {
		numWorkers = 4
	}

	chunkSize := totalKeys / int64(numWorkers)

	ctxCancel, cancel := context.WithCancel(ctx)
	defer cancel()

	type resultPayload struct {
		key       string
		plaintext string
	}
	resultChan := make(chan resultPayload, 1)
	var wg sync.WaitGroup
	var triedCount atomic.Int64

	// Key formatter based on mode
	formatKey := func(x int64) string {
		if mode == CrackModeHex {
			return fmt.Sprintf("%08x", x)
		}
		return fmt.Sprintf("%08d", x)
	}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			start := int64(workerID) * chunkSize
			end := start + chunkSize
			if workerID == numWorkers-1 {
				end = totalKeys
			}

			for x := start; x < end; x++ {
				select {
				case <-ctxCancel.Done():
					return
				default:
				}

				key := formatKey(x)
				crypto := NewTripleDESCrypto(key)

				decodedText, err := crypto.ECBDecrypt(authenticator)
				if err != nil {
					triedCount.Add(1)
					continue
				}

				infos := strings.Split(decodedText, "$")
				if len(infos) > 7 && infos[len(infos)-1] == "CTC" {
					select {
					case resultChan <- resultPayload{key: key, plaintext: decodedText}:
						cancel()
					default:
					}
					return
				}
				triedCount.Add(1)
			}
		}(i)
	}

	// Progress reporter goroutine
	var progressDone chan struct{}
	if progressCb != nil {
		progressDone = make(chan struct{})
		go func() {
			defer close(progressDone)
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctxCancel.Done():
					// Send final progress update
					tried := triedCount.Load()
					percent := float64(tried) / float64(totalKeys) * 100
					if percent > 100 {
						percent = 100
					}
					progressCb(CrackProgress{Tried: tried, Total: totalKeys, Percent: percent})
					return
				case <-ticker.C:
					tried := triedCount.Load()
					percent := float64(tried) / float64(totalKeys) * 100
					if percent > 100 {
						percent = 100
					}
					progressCb(CrackProgress{Tried: tried, Total: totalKeys, Percent: percent})
				}
			}
		}()
	}

	// Wait for all workers to complete, then close result channel
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	select {
	case r, ok := <-resultChan:
		cancel() // Signal progress goroutine to exit (may already be cancelled if key was found)
		if progressDone != nil {
			<-progressDone
		}
		if ok {
			return &CrackResult{
				Key:    r.key,
				Fields: parseAuthenticatorFields(r.plaintext),
			}, nil
		}
		return nil, errors.New("failed to crack the key in the search space")
	case <-ctx.Done():
		cancel()
		if progressDone != nil {
			<-progressDone
		}
		return nil, ctx.Err()
	}
}
