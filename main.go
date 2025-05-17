package main

import (
    "golang.org/x/tools/go/analysis/singlechecker"
    "analyzer/noreturn"
)

func main() {
    singlechecker.Main(noreturn.Analyzer)
}
