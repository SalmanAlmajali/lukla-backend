// Package main is the entry point for the optional background worker process.
// This binary is not needed until a reliable event dispatch use case is defined.
// See docs/architecture.md ADR-011 (Transactional Outbox) for context.
package main

import (
	"log"
)

func main() {
	// TODO: implement worker when outbox/scheduled job use cases are confirmed.
	log.Println("worker: not yet implemented")
}
