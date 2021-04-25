package main

import (
	"log"
	"os/exec"
	"strings"
	"testing"
)

/* BEGIN CONSTANTS */

/* END CONSTANTS */
/* BEGIN AUXILLARY FUNCTIONS */
func ExecCLI(CLIInput string) {
	args := strings.Split(CLIInput, " ")

	if len(args) < 2 {
		log.Fatal("Invalid Args to CLI")
	}

	cmd := exec.Command(args[0], args[1:]...)
	err := cmd.Run()

	if err != nil {
		log.Fatal(err)
	}
}

// func StartTestServer () {

// }

// func CloseTestServer () {

// }

/* END AUXILLARY FUNCTIONS */
/* BEGIN UNIT TESTING */

// Example Test:
// func TestXXX(t *testing.T) {
// 	// do testing
// }

func TestServerStart(t *testing.T) {

}

/* END UNIT TESTING */
