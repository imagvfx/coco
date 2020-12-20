package main

import (
	"errors"
	"flag"
)

func order(args []string) error {
	fset := flag.NewFlagSet("order", flag.ExitOnError)
	fset.Parse(args)
	fargs := fset.Args()
	if len(fargs) == 0 {
		return errors.New("need a json file to order")
	}
	return nil
}
