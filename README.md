# GoCalls

GoCalls is a static tool which produces a visual graph of the go statements in 
your Go program. It shows which functions/methods are called with the *go* statement.

For example the below program:

```
package main

type Widget struct {...}

func (w *Widget) Process() {...}

func (w *Widget) Consume() {...}

func run() {
    w := Widget{}
    go w.Process()
    go w.Consume()
}

func main() {
    go run()
    ...
}
```

Will produce the graph:

![graph](https://github.com/umahmood/gocalls/blob/master/graph.png)

This graph is showing that the *main* function makes a go call to *run*. The *run* 
function makes a go call to *Widget.Process* and *Widget.Consume*.  

# Installation

> $ go get github.com/umahmood/gocalls

# Usage

Processing a single go source code file:

> $ gocalls app.go <br/>
> Processing app.go ... <br/>
> Go statements: 3 <br/>
> Output: out.dot <br/>

GoCalls outputs a file ('out.dot') in the [dot language](http://www.graphviz.org/Documentation.php), which is the input into the dot graphviz tool.

> $ dot -Tpng out.dot -o graph.png

Processing a directory of go source files:

> $ gocalls $GOPATH/src/github.com/rakyll/ <br/>
> Processing boom.go ...<br/>
> Processing boom_test.go ...<br/>
> Processing boomer.go ...<br/>
> Processing boomer_test.go ...<br/>
> Processing print.go ...<br/>
> Processing copy.go ...<br/>
> Processing pb.go ...<br/>
> Processing format.go ...<br/>
> Processing pb.go ...<br/>
> Processing pb_nix.go ...<br/>
> Processing pb_win.go ...<br/>
> Go statements: 4<br/>
> Output: out.dot<br/>

# License

See the [LICENSE](LICENSE.md) file for license rights and limitations (MIT).
