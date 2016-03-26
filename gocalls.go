package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type funcCall struct {
	From string
	To   string
}

type compositeVisitor struct{}

var (
	funcCalls      = make([]funcCall, 0)
	compositeTypes = make(map[string]string)
	nGoStmts       int
	fset           *token.FileSet
)

// funcName returns the name of a function from a function declaration
func funcName(fn *ast.FuncDecl) string {
	if fn.Recv != nil {
		if fn.Recv.NumFields() > 0 {
			typ := fn.Recv.List[0].Type
			return fmt.Sprintf("(%s).%s", recvString(typ), fn.Name)
		}
	}
	return fn.Name.Name
}

// recvString formats the receiver of a function
func recvString(recv ast.Expr) string {
	switch t := recv.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + recvString(t.X)
	}
	return "BADRECV"
}

// containsGoStatemnt determines if a list of ast.Stmt's contains 'go' statements
func containsGoStatemnt(n []ast.Stmt) ([]funcCall, bool) {
	extractFuncName := func(g *ast.GoStmt) string {
		var fn string
		switch g := g.Call.Fun.(type) {
		case *ast.Ident:
			fn = g.Name
		case *ast.SelectorExpr:
			k := g.X.(*ast.Ident).Name
			// what is the type of k
			if v, ok := compositeTypes[k]; ok {
				k = "(" + v + ")."
			} else {
				k = k + "."
			}
			fn = k + g.Sel.Name
		case *ast.FuncLit:
			fn = formatAnonName(g)
		}
		return fn
	}
	var funcCalls []funcCall
	for i := 0; i < len(n); i++ {
		switch g := n[i].(type) {
		case *ast.RangeStmt:
			for i := 0; i < len(g.Body.List); i++ {
				if t, ok := g.Body.List[i].(*ast.GoStmt); ok {
					funcCalls = append(funcCalls, funcCall{To: extractFuncName(t)})
				}
			}
		case *ast.ForStmt:
			for i := 0; i < len(g.Body.List); i++ {
				if t, ok := g.Body.List[i].(*ast.GoStmt); ok {
					funcCalls = append(funcCalls, funcCall{To: extractFuncName(t)})
				}
			}
		case *ast.GoStmt:
			funcCalls = append(funcCalls, funcCall{To: extractFuncName(g)})
		}
	}
	if len(funcCalls) > 0 {
		return funcCalls, true
	}
	return nil, false
}

// formatAnonName formats the name of an anonymous function
func formatAnonName(n ast.Node) string {
	pos := fset.Position(n.Pos())
	fpath := filepath.Base(pos.Filename)
	return fmt.Sprintf("go func: %s:%d", fpath, pos.Line)
}

// Visit walks the AST tree and tries its best to determine the types of composite
// variables.
func (c *compositeVisitor) Visit(n ast.Node) ast.Visitor {
	switch n := n.(type) {
	case *ast.AssignStmt:
		if len(n.Lhs) != len(n.Rhs) {
			// e.g. a,b := foo()
			return c
		}
		for idx, vars := range n.Lhs {
			var v string

			switch vars := vars.(type) {
			case *ast.Ident:
				v = vars.Name
			case *ast.IndexExpr:
				// e.g. var["key"] = compositeType{}
				if x, ok := vars.X.(*ast.Ident); ok {
					v = x.Name
				}
			}
			if t, ok := n.Rhs[idx].(*ast.UnaryExpr); ok {
				if t.Op == token.AND {
					if c, ok := t.X.(*ast.CompositeLit); ok {
						switch c.Type.(type) {
						case *ast.Ident:
							compositeTypes[v] = c.Type.(*ast.Ident).Name
						case *ast.SelectorExpr:
							p := c.Type.(*ast.SelectorExpr).X.(*ast.Ident).Name
							compositeTypes[v] = p
						}
					}
				}
			}
			if t, ok := n.Rhs[idx].(*ast.CompositeLit); ok {
				if _, ok := t.Type.(*ast.Ident); ok {
					compositeTypes[v] = t.Type.(*ast.Ident).Name
				}
			}
		}
	}
	return c
}

// Visit walks the AST tree looking for go statements in function declarations
// and function literals.
func (fc *funcCall) Visit(n ast.Node) ast.Visitor {
	switch n := n.(type) {
	case *ast.FuncDecl:
		if t, ok := containsGoStatemnt(n.Body.List); ok {
			for _, loc := range t {
				funcCalls = append(funcCalls, funcCall{
					From: funcName(n),
					To:   loc.To,
				})
			}
		}
	case *ast.FuncLit:
		if t, ok := containsGoStatemnt(n.Body.List); ok {
			for _, loc := range t {
				funcCalls = append(funcCalls, funcCall{
					From: formatAnonName(n),
					To:   loc.To,
				})
			}
		}
	case *ast.GoStmt:
		nGoStmts++
	}
	return fc
}

// analyzeDir finds and returns a list of all go source code files in a
// directory.
func analyzeDir(dirname string) []string {
	var gofiles []string
	filepath.Walk(dirname, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(path, ".go") {
			gofiles = append(gofiles, path)
		}
		return err
	})
	return gofiles
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage:\ngocalls filename.go\ngocalls directory")
		os.Exit(1)
	}

	name := os.Args[1]

	fi, err := os.Stat(name)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var gofiles []string
	if fi.IsDir() {
		gofiles = analyzeDir(name)
	} else if strings.HasSuffix(name, ".go") {
		gofiles = append(gofiles, name)
	}

	if len(gofiles) == 0 {
		fmt.Println("gocalls: no *.go files found")
		os.Exit(1)
	}

	for _, file := range gofiles {
		fmt.Println("Processing", filepath.Base(file), "...")
		fset = token.NewFileSet()

		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			log.Fatal(err)
		}

		for _, decl := range f.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				ast.Walk(&compositeVisitor{}, fn)
				ast.Walk(&funcCall{}, fn)
			}
		}
	}

	if nGoStmts == 0 {
		fmt.Println("Go statements:", nGoStmts)
		os.Exit(0)
	}

	const fileName = "out.dot"

	file, err := os.Create(fileName)
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()

	// Crude!
	// Write output in the dot language
	io.WriteString(file, "digraph goflow {\n")

	for _, loc := range funcCalls {
		if loc.From == "main" {
			style := "\"main\" [shape=circle, style=filled, fillcolor=blue]\n"
			io.WriteString(file, style)
			break
		}
	}

	for _, loc := range funcCalls {
		n := fmt.Sprintf("\"%s\" -> \"%s\";\n", loc.From, loc.To)
		io.WriteString(file, n)
	}
	io.WriteString(file, "\n}\n")

	fmt.Println("Go statements:", nGoStmts)
	fmt.Println("Output:", fileName)
}
