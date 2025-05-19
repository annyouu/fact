package main

import (
    "golang.org/x/tools/go/analysis/singlechecker"
    "analyzer/paniccheck"
)

func main() {
    singlechecker.Main(paniccheck.Analyzer)
}
