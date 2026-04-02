package base

import (
	"log"
	"testing"
)

func TestCustomLogger(t *testing.T) {
	log.Printf("TestCustomLogger log.Printf")
	// Output: YYYY-mm-ddTHH:MM:SS.999+07:00 base_test.go:9: TestCustomLogger log.Printf

	t.Logf("TestCustomLogger t.Logf")
	// Output without timestamp
}
