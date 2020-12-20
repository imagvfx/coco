package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

func order(args []string) error {
	fset := flag.NewFlagSet("order", flag.ExitOnError)
	fset.Parse(args)
	fargs := fset.Args()
	if len(fargs) == 0 {
		return errors.New("need a json file to order")
	}
	orderfile := fargs[0]
	data, err := os.Open(orderfile)
	if err != nil {
		return err
	}
	cli := &http.Client{}
	addr := os.Getenv("COCO_ADDR")
	if addr == "" {
		addr = "localhost:8282"
	}
	req, err := http.NewRequest("POST", "http://"+addr+"/api/order", data)
	if err != nil {
		return err
	}
	resp, err := cli.Do(req)
	if err != nil {
		return err
	}
	fmt.Println(resp.Status)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(body))
	return nil
}
