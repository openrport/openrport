package main

import (
	"flag"
	"os"
	"text/template"
)

type data struct {
	Type string
	//   Name string
}

func main() {
	var d data
	flag.StringVar(&d.Type, "type", "", "The subtype used for the sort func being generated")
	// flag.StringVar(&d.Name, "name", "", "The name used for the queue being generated. This should start with a capital letter so that it is exported.")
	flag.Parse()

	t := template.Must(template.New("sorter").Parse(sortTemplate))
	t.Execute(os.Stdout, d)
}

var sortTemplate = `
package sorter



func (q *{{.Name}}) Dequeue() {{.Type}} {
  if q.list.Len() == 0 {
    panic(ErrEmptyQueue)
  }
  raw := q.list.Remove(q.list.Front())
  if typed, ok := raw.({{.Type}}); ok {
    return typed
  }
  panic(ErrInvalidType)
}
`
