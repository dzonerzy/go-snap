package main

import (
    "github.com/dzonerzy/go-snap/snap"
)

// Minimal wrapper: prefixes echo output while honoring flags like -n.
// Demonstrates ForwardUnknownFlags + TransformArgs ordering.
func main() {
    app := snap.New("echo-wrap", "prefix echo output")

    // Optional: expose a familiar -n flag on our wrapper too
    app.BoolFlag("n", "suppress trailing newline").Back()

    app.Wrap("/bin/echo").
        ForwardUnknownFlags().
        ForwardArgs().
        LeadingFlags("-n", "-e", "-E").
        MapBoolFlag("n", "-n").
        InsertAfterLeadingFlags("[prefix]").
        Passthrough().
        Back()

    app.RunAndExit()
}

// No custom split needed; the wrapper DSL LeadingFlags + InsertAfterLeadingFlags
// takes care of ordering.
