package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type ScriptIO struct {
	Subject    string   `json:"subject"`
	Recipients []string `json:"recipients"`
	Content    string   `json:"content"`
}

func main() {

	var tmp ScriptIO

	reader := bufio.NewReader(os.Stdin)
	all, err := io.ReadAll(reader)
	if err != nil {
		fmt.Println(err)
	}

	tmp.Content = string(all)
	tmp.Subject = os.Args[1]
	tmp.Recipients = os.Args[2:]

	serialized, err := json.Marshal(tmp)
	if err != nil {
		fmt.Println(err)
	}

	err = os.WriteFile("out.json", serialized, os.ModePerm)
	if err != nil {
		fmt.Println(err)
	}

}
