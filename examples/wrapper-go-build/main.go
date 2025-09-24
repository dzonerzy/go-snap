package main

import (
    "path/filepath"
    "strconv"
    "strings"

    "github.com/dzonerzy/go-snap/snap"
)

// Wrap `go build` and log every tool invocation via --toolexec.
// Usage:
//   mywrap build [flags passed through to `go build`]
// The go tool will invoke: `<self> log /path/to/tool <tool-args>` for each tool.
func main() {
    app := snap.New("mywrap", "Wrap go build and log tool invocations")

    // build command wraps `go build` and adds our toolexec shim
    app.Command("build", "wrap go build").
        Wrap("go").
        // Pass toolexec value as a single arg containing "<self> log"
        InjectArgsPre("build", "--toolexec", "${SELF} log").
        ForwardUnknownFlags(). // pass through any flags we don't define
        ForwardArgs().         // pass user args to `go build`
        Passthrough().
        Back()

    // dynamic shim invoked by the go tool via --toolexec
    app.Command("log", "toolexec logger").
        WrapDynamic().
        ForwardUnknownFlags(). // forward all tool flags (e.g., --V, -importcfg)
        // Optional: allow-list certain tools only
        // .AllowTools("asm","compile","vet","link")
        TransformArgs(func(ctx *snap.Context, in []string) ([]string, error) {
            if len(ctx.Args()) > 0 {
                tool := ctx.Args()[0]
                base := filepath.Base(tool)
                // quote args for clarity
                q := make([]string, len(in))
                for i, a := range in { q[i] = strconv.Quote(a) }
                ctx.Stderr().Write([]byte("[toolexec] " + base + " " + strings.Join(q, " ") + "\n"))
            }
            return in, nil
        }).
        Passthrough().
        HideFromHelp().
        Back()

    app.RunAndExit()
}
